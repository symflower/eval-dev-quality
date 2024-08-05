def magic_square_odd(base)
    if base.even? || base < 3
        raise ArgumentError, "base must be odd and > 2"
    end

    grid = Array.new(base) { Array.new(base, 0) }
    r, number = 0, 0
    size = base * base

    c = base / 2
    while number < size
        number += 1
        grid[r][c] = number
        if r == 0
            if c == base - 1
                r += 1
            else
                r = base - 1
                c += 1
            end
        else
            if c == base - 1
                r -= 1
                c = 0
            else
                if grid[r - 1][c + 1] == 0
                    r -= 1
                    c += 1
                else
                    r += 1
                end
            end
        end
    end

    return grid
end
