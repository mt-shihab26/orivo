use std::time::{Duration, Instant};

use ratatui::prelude::{Alignment, Buffer, Color, Line, Rect, Span, Stylize, Widget};

pub struct FpsProps {
    /// Smoothed frames-per-second over the last one-second interval.
    per_second: f64,
    /// Total number of frames rendered since the app started.
    per_lifetime: u64,
}

impl FpsProps {
    pub fn new() -> Self {
        Self {
            per_second: 0.0,
            per_lifetime: 0,
        }
    }
}

/// Stateful tracker that updates FPS props once per frame.
pub struct FpsState {
    props: FpsProps,
    /// Running frame count within the current one-second interval.
    frame_count_per_second: u32,
    /// Timestamp when the current one-second interval began.
    interval_start: Instant,
}

impl FpsState {
    pub fn new() -> Self {
        Self {
            props: FpsProps::new(),
            frame_count_per_second: 0,
            interval_start: Instant::now() - Duration::from_secs(1),
        }
    }

    pub fn props(&self) -> &FpsProps {
        &self.props
    }

    /// Call once per frame before rendering.
    pub fn tick(&mut self) {
        self.props.per_lifetime += 1;
        self.frame_count_per_second += 1;

        let elapsed = self.interval_start.elapsed().as_secs_f64();

        if elapsed >= 1.0 {
            self.props.per_second = self.frame_count_per_second as f64 / elapsed;
            self.frame_count_per_second = 0;
            self.interval_start = Instant::now();
        }
    }
}

/// Stateless widget that renders the FPS counter in the top-right corner.
pub struct FpsWidget<'a> {
    props: &'a FpsProps,
}

impl<'a> FpsWidget<'a> {
    pub fn new(props: &'a FpsProps) -> Self {
        Self { props }
    }
}

impl<'a> Widget for &FpsWidget<'a> {
    fn render(self, area: Rect, buf: &mut Buffer) {
        Line::from(
            Span::from(format!(
                "{:.0} fps  {} frames",
                self.props.per_second, self.props.per_lifetime
            ))
            .fg(Color::DarkGray),
        )
        .alignment(Alignment::Right)
        .render(area, buf);
    }
}
