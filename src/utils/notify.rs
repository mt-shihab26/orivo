use notify_rust::Notification;
use rodio::{DeviceSinkBuilder, Source, source::SineWave};
use std::{error::Error, thread, time::Duration};

use crate::{kinds::phase::Phase, log_error};

/// Escapes the markup the freedesktop notification spec allows in bodies
/// (`<b>`, `<i>`, `<a>`, `<img>`), so todo text is shown literally instead of
/// being interpreted by the notification daemon.
fn escape_markup(text: &str) -> String {
    text.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
}

/// Sends a desktop notification with the app name prepended to the summary.
pub fn notify(summary: &str, body: &str, phase: &Phase) {
    let summary = format!("{} — {summary}", env!("CARGO_PKG_NAME"));
    if let Err(e) = Notification::new()
        .summary(&summary)
        .body(&escape_markup(body))
        .sound_name("message-new-instant")
        .show()
    {
        log_error!("failed to send notification: {e}");
    }
    sound(phase);
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn escapes_markup_in_todo_text() {
        assert_eq!(
            escape_markup("<img src=\"x\"> & <b>bold</b>"),
            "&lt;img src=\"x\"&gt; &amp; &lt;b&gt;bold&lt;/b&gt;"
        );
    }

    #[test]
    fn leaves_ordinary_text_alone() {
        assert_eq!(escape_markup("Work on PaystubHero"), "Work on PaystubHero");
    }
}

fn sound(phase: &Phase) {
    let (freq, duration_ms) = match phase {
        // Work done → warm, satisfying tone
        Phase::Work => (660.0_f32, 150_u64),
        // Short break done → bright, alerting tone
        Phase::Break => (880.0_f32, 100_u64),
        // Long break done → softer, lower tone
        Phase::LongBreak => (523.0_f32, 200_u64),
    };

    thread::spawn(move || {
        let result = (|| -> Result<(), Box<dyn Error>> {
            let mut stream_handle = DeviceSinkBuilder::open_default_sink()?;
            stream_handle.log_on_drop(false);
            let mixer = stream_handle.mixer();
            let wave = SineWave::new(freq)
                .amplify(0.15)
                .take_duration(Duration::from_millis(duration_ms));
            mixer.add(wave);
            thread::sleep(Duration::from_millis(duration_ms + 30));
            Ok(())
        })();
        if let Err(e) = result {
            log_error!("failed to play notification sound: {e}");
        }
    });
}
