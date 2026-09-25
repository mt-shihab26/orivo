use time::{Date, Month, OffsetDateTime, PrimitiveDateTime, Time};

/// Returns the current local time, falling back to UTC if the local offset is unavailable.
pub fn now() -> OffsetDateTime {
    OffsetDateTime::now_local().unwrap_or_else(|_| OffsetDateTime::now_utc())
}

/// Returns today's local date, falling back to UTC if the local offset is unavailable.
pub fn today() -> Date {
    now().date()
}

/// Formats a date as `"YYYY-MM-DD"`.
pub fn format_date(date: Date) -> String {
    format!(
        "{}-{:02}-{:02}",
        date.year(),
        date.month() as u8,
        date.day()
    )
}

/// Parses a `"YYYY-MM-DD"` or `"YYYY-MM-DDTHH:MM:SSZ"` string into a `Date`. Only the first 10
/// characters (the date portion) are used.
pub fn parse_date(s: &str) -> Option<Date> {
    let date_part = &s[..s.len().min(10)];
    let mut parts = date_part.splitn(3, '-');
    let year: i32 = parts.next()?.parse().ok()?;
    let month: u8 = parts.next()?.parse().ok()?;
    let day: u8 = parts.next()?.parse().ok()?;
    Date::from_calendar_date(year, Month::try_from(month).ok()?, day).ok()
}

/// Formats an `OffsetDateTime` for people to read, e.g. `"Fri, 25 Sep 2026, 8:02 PM (UTC+06:00)"`.
pub fn format_human_datetime(dt: OffsetDateTime) -> String {
    let (hour, meridiem) = match dt.hour() {
        0 => (12, "AM"),
        h @ 1..=11 => (h, "AM"),
        12 => (12, "PM"),
        h => (h - 12, "PM"),
    };
    let offset = dt.offset();
    let (_, om, _) = offset.as_hms();
    let sign = if offset.is_negative() { '-' } else { '+' };
    format!(
        "{:.3}, {} {:.3} {}, {}:{:02} {} (UTC{}{:02}:{:02})",
        dt.weekday().to_string(),
        dt.day(),
        dt.month().to_string(),
        dt.year(),
        hour,
        dt.minute(),
        meridiem,
        sign,
        offset.whole_hours().unsigned_abs(),
        om.unsigned_abs(),
    )
}

/// Formats an `OffsetDateTime` as `"YYYY-MM-DDTHH:MM:SS±HH:MM"`.
pub fn format_datetime(dt: OffsetDateTime) -> String {
    let offset = dt.offset();
    let (oh, om, _) = offset.as_hms();
    let sign = if oh >= 0 { '+' } else { '-' };
    format!(
        "{}-{:02}-{:02}T{:02}:{:02}:{:02}{}{:02}:{:02}",
        dt.year(),
        dt.month() as u8,
        dt.day(),
        dt.hour(),
        dt.minute(),
        dt.second(),
        sign,
        oh.unsigned_abs(),
        om.unsigned_abs(),
    )
}

/// Parses an ISO 8601 datetime string into an `OffsetDateTime`, falling back to local time on
/// failure. Handles `"YYYY-MM-DDTHH:MM:SS±HH:MM"`, `"YYYY-MM-DDTHH:MM:SSZ"`, and `"YYYY-MM-DD"`.
pub fn parse_datetime(s: &str) -> Option<OffsetDateTime> {
    if s.len() >= 19 {
        let date = parse_date(&s[..10])?;
        let hour: u8 = s[11..13].parse().ok()?;
        let minute: u8 = s[14..16].parse().ok()?;
        let second: u8 = s[17..19].parse().ok()?;
        let time = Time::from_hms(hour, minute, second).ok()?;
        let offset = if s.len() >= 25 && (s.as_bytes()[19] == b'+' || s.as_bytes()[19] == b'-') {
            let sign: i8 = if s.as_bytes()[19] == b'+' { 1 } else { -1 };
            let oh: i8 = s[20..22].parse::<i8>().ok()? * sign;
            let om: i8 = s[23..25].parse::<i8>().ok()?;
            time::UtcOffset::from_hms(oh, om, 0).ok()?
        } else {
            time::UtcOffset::UTC
        };
        Some(PrimitiveDateTime::new(date, time).assume_offset(offset))
    } else {
        let local = now();
        Some(PrimitiveDateTime::new(parse_date(s)?, Time::MIDNIGHT).assume_offset(local.offset()))
    }
}

/// Shifts a date by `delta` months, clamping the day to the last valid day of the target month.
pub fn shift_month(date: Date, delta: i32) -> Date {
    let total = date.month() as i32 - 1 + delta;
    let new_year = date.year() + total.div_euclid(12);
    let new_month_num = (total.rem_euclid(12) + 1) as u8;
    if let Ok(m) = Month::try_from(new_month_num) {
        let new_day = date.day().min(days_in_month(new_year, m));
        if let Ok(d) = Date::from_calendar_date(new_year, m, new_day) {
            return d;
        }
    }
    date
}

/// Returns the number of days in the given month of the given year.
fn days_in_month(year: i32, month: Month) -> u8 {
    let (ny, nm) = if month == Month::December {
        (year + 1, 1u8)
    } else {
        (year, month as u8 + 1)
    };
    if let Ok(m) = Month::try_from(nm) {
        if let Ok(first) = Date::from_calendar_date(ny, m, 1) {
            if let Some(last) = first.previous_day() {
                return last.day();
            }
        }
    }
    28
}

#[cfg(test)]
mod tests {
    use super::*;
    use time::Month;

    #[test]
    fn format_date_pads_month_and_day() {
        let date = Date::from_calendar_date(2024, Month::January, 5).unwrap();
        assert_eq!(format_date(date), "2024-01-05");
    }

    #[test]
    fn parse_date_full() {
        let date = parse_date("2024-03-15").unwrap();
        assert_eq!(date.year(), 2024);
        assert_eq!(date.month(), Month::March);
        assert_eq!(date.day(), 15);
    }

    #[test]
    fn parse_date_from_datetime_string() {
        let date = parse_date("2024-03-15T12:00:00Z").unwrap();
        assert_eq!(date.year(), 2024);
        assert_eq!(date.month(), Month::March);
        assert_eq!(date.day(), 15);
    }

    #[test]
    fn parse_date_invalid_returns_none() {
        assert!(parse_date("not-a-date").is_none());
    }

    #[test]
    fn format_datetime_positive_offset() {
        let date = Date::from_calendar_date(2024, Month::June, 1).unwrap();
        let time = Time::from_hms(10, 30, 0).unwrap();
        let offset = time::UtcOffset::from_hms(5, 30, 0).unwrap();
        let dt = PrimitiveDateTime::new(date, time).assume_offset(offset);
        assert_eq!(format_datetime(dt), "2024-06-01T10:30:00+05:30");
    }

    #[test]
    fn format_datetime_utc() {
        let date = Date::from_calendar_date(2024, Month::January, 1).unwrap();
        let time = Time::from_hms(0, 0, 0).unwrap();
        let dt = PrimitiveDateTime::new(date, time).assume_utc();
        assert_eq!(format_datetime(dt), "2024-01-01T00:00:00+00:00");
    }

    #[test]
    fn format_human_datetime_evening_positive_offset() {
        let date = Date::from_calendar_date(2026, Month::September, 25).unwrap();
        let time = Time::from_hms(20, 2, 3).unwrap();
        let offset = time::UtcOffset::from_hms(6, 0, 0).unwrap();
        let dt = PrimitiveDateTime::new(date, time).assume_offset(offset);
        assert_eq!(
            format_human_datetime(dt),
            "Fri, 25 Sep 2026, 8:02 PM (UTC+06:00)"
        );
    }

    #[test]
    fn format_human_datetime_midnight_and_noon() {
        let date = Date::from_calendar_date(2024, Month::January, 1).unwrap();
        let midnight = PrimitiveDateTime::new(date, Time::from_hms(0, 5, 0).unwrap()).assume_utc();
        let noon = PrimitiveDateTime::new(date, Time::from_hms(12, 0, 0).unwrap()).assume_utc();
        assert_eq!(
            format_human_datetime(midnight),
            "Mon, 1 Jan 2024, 12:05 AM (UTC+00:00)"
        );
        assert_eq!(
            format_human_datetime(noon),
            "Mon, 1 Jan 2024, 12:00 PM (UTC+00:00)"
        );
    }

    #[test]
    fn format_human_datetime_negative_offsets() {
        let date = Date::from_calendar_date(2024, Month::March, 5).unwrap();
        let time = Time::from_hms(9, 15, 0).unwrap();
        let ny = time::UtcOffset::from_hms(-5, 0, 0).unwrap();
        let newfoundland = time::UtcOffset::from_hms(-3, -30, 0).unwrap();
        let dt = PrimitiveDateTime::new(date, time);
        assert_eq!(
            format_human_datetime(dt.assume_offset(ny)),
            "Tue, 5 Mar 2024, 9:15 AM (UTC-05:00)"
        );
        assert_eq!(
            format_human_datetime(dt.assume_offset(newfoundland)),
            "Tue, 5 Mar 2024, 9:15 AM (UTC-03:30)"
        );
    }

    #[test]
    fn parse_datetime_with_offset() {
        let dt = parse_datetime("2024-06-01T10:30:00+05:30").unwrap();
        assert_eq!(dt.year(), 2024);
        assert_eq!(dt.hour(), 10);
        assert_eq!(dt.minute(), 30);
    }

    #[test]
    fn parse_datetime_utc_z() {
        let dt = parse_datetime("2024-01-15T08:00:00Z").unwrap();
        assert_eq!(dt.offset(), time::UtcOffset::UTC);
        assert_eq!(dt.day(), 15);
    }

    #[test]
    fn parse_datetime_date_only() {
        let dt = parse_datetime("2024-03-20").unwrap();
        assert_eq!(dt.year(), 2024);
        assert_eq!(dt.month(), Month::March);
        assert_eq!(dt.day(), 20);
        assert_eq!(dt.hour(), 0);
    }

    #[test]
    fn shift_month_forward() {
        let date = Date::from_calendar_date(2024, Month::January, 15).unwrap();
        let result = shift_month(date, 2);
        assert_eq!(result.month(), Month::March);
        assert_eq!(result.day(), 15);
    }

    #[test]
    fn shift_month_crosses_year() {
        let date = Date::from_calendar_date(2024, Month::November, 10).unwrap();
        let result = shift_month(date, 3);
        assert_eq!(result.year(), 2025);
        assert_eq!(result.month(), Month::February);
    }

    #[test]
    fn shift_month_clamps_day_to_month_end() {
        // Jan 31 + 1 month = Feb 29 (2024 is a leap year)
        let date = Date::from_calendar_date(2024, Month::January, 31).unwrap();
        let result = shift_month(date, 1);
        assert_eq!(result.month(), Month::February);
        assert_eq!(result.day(), 29);
    }

    #[test]
    fn shift_month_backward() {
        let date = Date::from_calendar_date(2024, Month::March, 15).unwrap();
        let result = shift_month(date, -1);
        assert_eq!(result.month(), Month::February);
        assert_eq!(result.day(), 15);
    }
}
