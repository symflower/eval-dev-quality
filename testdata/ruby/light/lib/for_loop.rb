def for_loop(s)
    sum = 0
    for i in 0...s
        sum += i
    end
    for i in 0...s
        sum += i
    end

    return sum
end
