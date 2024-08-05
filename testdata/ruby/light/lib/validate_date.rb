def valid_date(day, month, year)
    month_days = [31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31]

    if year < 1583
        return false
    end
    if month < 1 || month > 12
        return false
    end
    if day < 1
        return false
    end
    if month == 2
        if (year % 400) != 0 && (year % 4) == 0
            if day > 29
                return false
            end
        else
            if day > 28
                return false
            end
        end
    else
        if day > month_days[month - 1]
            return false
        end
    end

    return true
end
