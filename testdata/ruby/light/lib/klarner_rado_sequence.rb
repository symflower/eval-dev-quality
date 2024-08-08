def initialise_klarner_rado_sequence(limit)
    result = Array.new(limit + 1)
    i2, i3 = 1, 1
    m2, m3 = 1, 1

    (1..limit).each do |i|
        minimum = [m2, m3].min
        result[i] = minimum
        if m2 == minimum
            m2 = result[i2] * 2 + 1
            i2 += 1
        end
        if m3 == minimum
            m3 = result[i3] * 3 + 1
            i3 += 1
        end
    end

    return result
end
