def type_unknown(x)
    if x.is_a?(Intt)
        if x > 0
            return 1
        elsif x < 0
            return -1
        end
    end

    return 0
end
