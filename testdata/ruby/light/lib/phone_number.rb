class ExtractDigitsError < StandardError; end

def extract_digits(dirty_number)
    clean_number = ""

    dirty_number.each_char do |c|
        if c == ' ' || c == '.' || c == '(' || c == ')' || c == '-' || c == '+'
            # Remove spaces, dots, parentheses, hyphens, and pluses.
            next
        end
        if c == '-' || c == '@' || c == ':' || c == '!'
            raise ExtractDigitsError, "Punctuations not permitted"
        end
        if c < '0' || c > '9'
            raise ExtractDigitsError, "Letters not permitted"
        end
        clean_number += c
    end

    return clean_number
end
