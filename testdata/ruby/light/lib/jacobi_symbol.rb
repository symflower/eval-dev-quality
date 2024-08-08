def jacobi_symbol(k, n)
    if k < 0 || n.even?
        raise ArgumentError, "Invalid value. k = #{k}, n = #{n}"
    end
    k %= n
    jacobi = 1
    while k > 0
        while k.even?
            k /= 2
            r = n % 8
            if r == 3 || r == 5
                jacobi = -jacobi
            end
        end
        temp = n
        n = k
        k = temp
        if k % 4 == 3 && n % 4 == 3
            jacobi = -jacobi
        end
        k %= n
    end

    if n == 1
        return jacobi
    end

    return 0
end
