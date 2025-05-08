pub fn has_balanced_brackets(char_array: &str) -> bool {
    let mut brackets = 0;

    for ch in char_array.chars() {
        if ch == '[' {
            brackets += 1;
        } else if ch == ']' {
            brackets -= 1;
        } else {
            return false; // Non-bracket characters
        }

        if brackets < 0 { // Closing bracket before opening bracket
            return false;
        }
    }

    brackets == 0
}
