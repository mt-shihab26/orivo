use std::{
    io::{Error, ErrorKind, Result},
    sync::{
        Arc, Mutex,
        mpsc::{self, Receiver, RecvTimeoutError},
    },
    time::Duration,
};

use sea_orm::DatabaseConnection;

use ratatui::{
    DefaultTerminal, Frame,
    crossterm::{
        event::{
            DisableMouseCapture, EnableMouseCapture, KeyCode, KeyModifiers, MouseButton,
            MouseEventKind,
        },
        execute,
    },
    init,
    prelude::{Color, Constraint, Layout, Line, Rect, Span, Stylize, Widget},
    restore,
    style::Style,
    widgets::{Block, Paragraph},
};

use crate::{
    caches::timer::TimerCache,
    config::Config,
    kinds::event::Event,
    log_error, log_info,
    tabs::{Tab, timer::TimerTab, todos::TodosTab},
    utils::store::Store,
    widgets::layout::fps::{FpsState, FpsWidget},
    workers::term,
};

/// The root application, owning all tabs and driving the main event loop.
pub struct App {
    alive: bool,
    selected: usize,
    tabs: Vec<Box<dyn Tab>>,
    events: Receiver<Event>,
    /// FPS counter state, `None` when disabled.
    fps_state: Option<FpsState>,
    /// Bounding rects of each tab header, updated every frame for click detection.
    tab_rects: Vec<Rect>,
}

impl App {
    /// Creates a new `App`, spawning the terminal event worker and initialising all tabs.
    pub fn new(config: Config, db: DatabaseConnection) -> Self {
        let (sender, events) = mpsc::channel::<Event>();

        term::spawn(sender.clone());

        let timer_cache = Arc::new(Mutex::new(TimerCache::new(db.clone())));
        let store = Store::load();

        Self {
            alive: true,
            selected: 1,
            tabs: vec![
                Box::new(TodosTab::new(db.clone(), Arc::clone(&timer_cache))),
                Box::new(TimerTab::new(sender, db, config.timer, timer_cache, store)),
            ],
            events,
            fps_state: config.show_fps.then(FpsState::new),
            tab_rects: Vec::new(),
        }
    }

    /// Initialises the terminal, runs the event loop, then restores the terminal on exit.
    pub fn run(&mut self) -> Result<()> {
        let mut terminal = init();
        execute!(std::io::stdout(), EnableMouseCapture)?;

        let result = self.event_loop(&mut terminal);

        execute!(std::io::stdout(), DisableMouseCapture)?;
        restore();

        result
    }

    /// Drives the main loop: tick FPS, draw, receive event, handle event.
    fn event_loop(&mut self, terminal: &mut DefaultTerminal) -> Result<()> {
        while self.alive {
            self.tick_fps();
            self.terminal_draw(terminal)?;
            let event = self.recv_event()?;
            self.handle_event(event)?;
        }

        Ok(())
    }

    fn tick_fps(&mut self) {
        if let Some(fps_state) = &mut self.fps_state {
            fps_state.tick();
        }
    }

    fn terminal_draw(&mut self, terminal: &mut DefaultTerminal) -> Result<()> {
        if let Err(e) = terminal.draw(|frame| self.render_frame(frame)) {
            log_error!("terminal draw failed: {e}");
            return Err(e);
        }
        Ok(())
    }

    /// Renders the full UI: app bar, tab headers, and the active tab's content.
    fn render_frame(&mut self, frame: &mut Frame) {
        let area = frame.area();
        let buf = frame.buffer_mut();
        let active_tab = &self.tabs[self.selected];

        let [top, tabs_header, tab_content] = Layout::vertical([
            Constraint::Length(1),
            Constraint::Length(3),
            Constraint::Fill(1),
        ])
        .areas(area);

        Paragraph::new(
            Span::from(self.get_app_name())
                .bold()
                .fg(active_tab.color()),
        )
        .centered()
        .render(top, buf);

        let [top_left, top_right] =
            Layout::horizontal([Constraint::Fill(1), Constraint::Fill(1)]).areas(top);

        Line::from(self.get_hints()).render(top_left, buf);

        if let Some(fps_state) = &self.fps_state {
            FpsWidget::new(fps_state.props()).render(top_right, buf);
        }

        let tab_areas =
            Layout::horizontal(vec![Constraint::Fill(1); self.tabs.len()]).split(tabs_header);
        self.tab_rects = tab_areas.to_vec();

        for index in 0..self.tabs.len() {
            let color = self.get_tab_color(index);

            let block = Block::bordered().border_style(Style::default().fg(color).bold());
            let inner = block.inner(tab_areas[index]);
            block.render(tab_areas[index], buf);

            Paragraph::new(self.tabs[index].name())
                .centered()
                .style(Style::default().fg(color).bold())
                .render(inner, buf);
        }

        active_tab.render(frame, tab_content);
    }

    fn get_app_name(&self) -> &str {
        "Orivo"
    }

    /// Returns the global keybinding hints shown in the top-left corner.
    fn get_hints(&self) -> Vec<Span<'static>> {
        let mut hints = vec![
            Span::from("^q").fg(Color::DarkGray).bold(),
            Span::from(" quit").fg(Color::DarkGray),
            Span::from("  ").fg(Color::DarkGray),
            Span::from("^r").fg(Color::DarkGray).bold(),
            Span::from(" reload").fg(Color::DarkGray),
        ];

        if self.fps_state.is_some() {
            hints.push(Span::from("  ").fg(Color::DarkGray));
            hints.push(Span::from("^f").fg(Color::DarkGray).bold());
            hints.push(Span::from(" fps").fg(Color::DarkGray));
        }

        hints
    }

    /// Returns the accent color for the tab at `index` — active color or dimmed gray.
    fn get_tab_color(&self, index: usize) -> Color {
        if index == self.selected {
            self.tabs[index].color()
        } else {
            Color::DarkGray
        }
    }

    /// Receives the next event, blocking or timing out depending on whether the active tab needs ticking.
    fn recv_event(&self) -> Result<Option<Event>> {
        let event = if self.tabs[self.selected].should_tick() {
            match self.events.recv_timeout(Duration::from_millis(8)) {
                Ok(event) => Some(event),
                Err(RecvTimeoutError::Timeout) => None,
                Err(RecvTimeoutError::Disconnected) => {
                    log_error!("event channel disconnected");
                    return Err(Error::new(
                        ErrorKind::BrokenPipe,
                        "event channel disconnected",
                    ));
                }
            }
        } else {
            match self.events.recv() {
                Ok(event) => Some(event),
                Err(e) => {
                    log_error!("event channel disconnected: {e}");
                    return Err(Error::new(ErrorKind::BrokenPipe, e));
                }
            }
        };

        Ok(event)
    }

    fn handle_event(&mut self, event: Option<Event>) -> Result<()> {
        let ctrl = |key: &ratatui::crossterm::event::KeyEvent| {
            key.modifiers.contains(KeyModifiers::CONTROL)
        };

        match event {
            Some(Event::Key(key)) => match key.code {
                KeyCode::Char('c' | 'q') if ctrl(&key) => self.quit(),
                KeyCode::Char('f') if ctrl(&key) => self.toggle_fps(),
                KeyCode::Char('r') if ctrl(&key) => self.invalidate_caches(),
                KeyCode::Char('t') if ctrl(&key) => self.select_tab(0),
                KeyCode::Char('y') if ctrl(&key) => self.select_tab(1),
                KeyCode::Tab => self.next_tab(),
                _ => self.handle_key(key)?,
            },
            Some(Event::Mouse(mouse)) => {
                if mouse.kind == MouseEventKind::Down(MouseButton::Left) {
                    for (i, rect) in self.tab_rects.iter().enumerate() {
                        if mouse.column >= rect.x
                            && mouse.column < rect.x + rect.width
                            && mouse.row >= rect.y
                            && mouse.row < rect.y + rect.height
                        {
                            self.select_tab(i);
                            break;
                        }
                    }
                }
            }
            Some(Event::Resize(width, height)) => {
                log_info!("resized width: {width}, height: {height}")
            }
            Some(Event::TimerTick) => {}
            None => self.tick_tab()?,
        }

        Ok(())
    }

    fn quit(&mut self) {
        self.alive = false;
    }

    fn toggle_fps(&mut self) {
        self.fps_state = if self.fps_state.is_none() {
            Some(FpsState::new())
        } else {
            None
        };
    }

    fn invalidate_caches(&mut self) {
        for tab in &mut self.tabs {
            tab.invalidate_cache();
        }
    }

    fn select_tab(&mut self, index: usize) {
        self.selected = index;
    }

    fn next_tab(&mut self) {
        self.selected = (self.selected + 1) % self.tabs.len();
    }

    fn handle_key(&mut self, key: ratatui::crossterm::event::KeyEvent) -> Result<()> {
        self.tabs[self.selected].handle(key).map_err(|e| {
            log_error!("tab handle error: {e}");
            e
        })
    }

    /// Calls `next_tick` on the active tab (used when no event arrived within the timeout).
    fn tick_tab(&mut self) -> Result<()> {
        self.tabs[self.selected].next_tick().map_err(|e| {
            log_error!("tab tick error: {e}");
            e
        })
    }
}
