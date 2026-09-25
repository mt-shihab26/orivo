use time::{Date, Duration};

/// A recurrence rule for a todo item.
#[derive(Clone, Debug, PartialEq)]
pub enum Repeat {
    Daily,
    /// Repeats on the same weekday each week.
    Weekly,
    /// Repeats on weekends only (Saturday–Sunday).
    Weekends,
    /// Repeats on weekdays only (Monday–Friday).
    Weekdays,
    /// Repeats on the same day of each month.
    Monthly,
    /// Repeats on the same day each year.
    Yearly,
}

impl Repeat {
    pub fn icon() -> &'static str {
        "⟳"
    }

    /// All repeat variants in display order.
    pub const ALL: &'static [Repeat] = &[
        Repeat::Daily,
        Repeat::Weekly,
        Repeat::Weekends,
        Repeat::Weekdays,
        Repeat::Monthly,
        Repeat::Yearly,
    ];

    /// Clones a `Repeat` value from a reference.
    pub fn of(r: &Repeat) -> Repeat {
        match r {
            Repeat::Daily => Repeat::Daily,
            Repeat::Weekly => Repeat::Weekly,
            Repeat::Weekends => Repeat::Weekends,
            Repeat::Weekdays => Repeat::Weekdays,
            Repeat::Monthly => Repeat::Monthly,
            Repeat::Yearly => Repeat::Yearly,
        }
    }

    pub fn label(&self) -> &str {
        match self {
            Repeat::Daily => "Daily",
            Repeat::Weekly => "Weekly (same day)",
            Repeat::Weekends => "Weekends (Sat-Sun)",
            Repeat::Weekdays => "Weekdays (Mon-Fri)",
            Repeat::Monthly => "Monthly on day",
            Repeat::Yearly => "Yearly on day",
        }
    }

    pub fn to_db_str(&self) -> &str {
        match self {
            Repeat::Daily => "daily",
            Repeat::Weekly => "weekly_same_day",
            Repeat::Weekends => "weekends_sat_sun",
            Repeat::Weekdays => "weekdays_mon_fri",
            Repeat::Monthly => "monthly_on_day",
            Repeat::Yearly => "yearly_on_day",
        }
    }

    pub fn from_db_str(s: &str) -> Option<Self> {
        match s {
            "daily" => Some(Repeat::Daily),
            "weekly_same_day" => Some(Repeat::Weekly),
            "weekends_sat_sun" => Some(Repeat::Weekends),
            "weekdays_mon_fri" => Some(Repeat::Weekdays),
            "monthly_on_day" => Some(Repeat::Monthly),
            "yearly_on_day" => Some(Repeat::Yearly),
            _ => None,
        }
    }

    /// Next due date after `from`.
    pub fn next_date(&self, from: Date) -> Date {
        match self {
            Repeat::Daily => from + Duration::days(1),
            Repeat::Weekly => from + Duration::weeks(1),
            Repeat::Weekends => {
                let mut next = from + Duration::days(1);
                while matches!(
                    next.weekday(),
                    time::Weekday::Monday
                        | time::Weekday::Tuesday
                        | time::Weekday::Wednesday
                        | time::Weekday::Thursday
                        | time::Weekday::Friday
                ) {
                    next = next + Duration::days(1);
                }
                next
            }
            Repeat::Weekdays => {
                let mut next = from + Duration::days(1);
                while matches!(
                    next.weekday(),
                    time::Weekday::Saturday | time::Weekday::Sunday
                ) {
                    next = next + Duration::days(1);
                }
                next
            }
            Repeat::Monthly => {
                let month = from.month().next();
                let year = if month == time::Month::January {
                    from.year() + 1
                } else {
                    from.year()
                };
                Date::from_calendar_date(year, month, from.day()).unwrap_or(from)
            }
            Repeat::Yearly => {
                Date::from_calendar_date(from.year() + 1, from.month(), from.day()).unwrap_or(from)
            }
        }
    }
}
