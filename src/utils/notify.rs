use notify_rust::Notification;
use rodio::{DeviceSinkBuilder, Source, source::SineWave};
use std::{error::Error, thread, time::Duration};

use crate::log_error;

/// Escapes the markup the freedesktop notification spec allows in bodies
/// (`<b>`, `<i>`, `<a>`, `<img>`), so todo text is shown literally instead of
/// being interpreted by the notification daemon.
fn escape_markup(text: &str) -> String {
    text.replace('&', "&amp;")
        .replace('<', "&lt;")
        .replace('>', "&gt;")
}

/// Sends a desktop notification with the app name prepended to the summary, playing the
/// freedesktop notification sound plus a synthesized alert tone.
pub fn notify(summary: &str, body: &str) {
    send(summary, body, Some("message-new-instant"));
    sound();
}

/// Sends a desktop notification with no sound at all — for background events (e.g. sync)
/// where an alert tone would be unwarranted.
pub fn notify_silent(summary: &str, body: &str) {
    send(summary, body, None);
}

/// Shows the notification itself, with the app name prepended to the summary and an optional
/// freedesktop sound name.
fn send(summary: &str, body: &str, sound_name: Option<&str>) {
    let summary = format!("{} — {summary}", env!("CARGO_PKG_NAME"));
    let mut notification = Notification::new();
    notification.summary(&summary).body(&escape_markup(body));
    if let Some(name) = sound_name {
        notification.sound_name(name);
    }
    if let Err(e) = notification.show() {
        log_error!("failed to send notification: {e}");
    }
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

/// Plays the same bright, alerting tone for every phase transition.
fn sound() {
    let (freq, duration_ms) = (880.0_f32, 100_u64);

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
