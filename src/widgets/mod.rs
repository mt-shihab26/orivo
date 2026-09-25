//! # Widget pattern
//!
//! Every UI component is split into **Action? → Props → State? → Widget**
//! (see [`crate::widgets::timer::todo_picker`] for a full example):
//!
//! - **Action** *(optional)*: enum returned by `State::handle(key)` to signal events
//!   (select, cancel, …) to the caller. Omit for purely visual widgets.
//! - **Props**: plain data needed to render one frame. The widget only borrows it.
//! - **State** *(optional)*: owns the props that change over time and lives in the caller.
//!   Exposes `props()` for rendering and optionally `handle()`. Stateless widgets skip it
//!   and the caller builds `Props` directly.
//! - **Widget**: a stateless view created at render time and dropped afterwards.
//!
//! ```rust,ignore
//! match self.my_state.handle(key) {
//!     MyAction::Select(item) => { /* ... */ }
//!     MyAction::Cancel => { /* dismiss */ }
//!     MyAction::None => {}
//! }
//! MyWidget::new(self.my_state.props()).render(area, buf);
//! ```
//!
//! Rules: state is never passed to a widget (only `&props`); widgets are never stored;
//! visibility is the caller's concern (wrap the render call in an `if`); implement
//! `Widget for &MyWidget`.

pub mod layout;
pub mod timer;
pub mod todos;
