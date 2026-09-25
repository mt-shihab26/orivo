/// A view within the todos tab.
#[derive(Clone, Copy, PartialEq, Eq)]
pub enum Page {
    /// Todos that are past due.
    Due,
    Today,
    /// All todos grouped by date.
    Index,
    /// Completed todos.
    History,
}

impl Page {
    /// All pages in display order.
    pub const ALL: &'static [Page] = &[Page::Due, Page::Today, Page::Index, Page::History];

    /// Returns the display label for the page, including its keybinding.
    pub fn label(&self) -> &str {
        match self {
            Page::Due => "Due [1]",
            Page::Today => "Today [2]",
            Page::Index => "Index [3]",
            Page::History => "History [4]",
        }
    }

    pub fn index(&self) -> usize {
        Self::ALL.iter().position(|p| p == self).unwrap()
    }

    /// Returns the next page, wrapping around.
    pub fn next(&self) -> Page {
        match self {
            Page::Due => Page::Today,
            Page::Today => Page::Index,
            Page::Index => Page::History,
            Page::History => Page::Due,
        }
    }

    /// Returns the previous page, wrapping around.
    pub fn prev(&self) -> Page {
        match self {
            Page::Due => Page::History,
            Page::Today => Page::Due,
            Page::Index => Page::Today,
            Page::History => Page::Index,
        }
    }
}
