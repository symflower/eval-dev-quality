def largest_proper_divisor(a_number)
    if a_number < 1
        raise ArgumentError, "Argument must be >= 1: #{a_number}"
    end

    if (a_number & 1) == 0
        return a_number >> 1
    end

    p = 3
    while p * p <= a_number
        if a_number % p == 0
            return a_number / p
        end
        p += 2
    end

    return 1
end
