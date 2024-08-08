def match_bytes(s1, s2)
    s1.each_index do |i|
        c1 = s1[i]
        c2 = s2[i]

        if c1 != c2
            c1 |= ('a'.ord - 'A'.ord)
            c2 |= ('a'.ord - 'A'.ord)

            if c1 != c2 || c1 < 'a'.ord || c1 > 'z'.ord
                return false
            end
        end
    end

    return true
end
