def compute_step_count(start)
    if start <= 0
        raise ArgumentError, "Only positive integers are allowed"
    end
    if start == 1
        return 0
    end

    next_value = if start.even?
        start / 2
    else
        3 * start + 1
    end

    return 1 + compute_step_count(next_value)
end
