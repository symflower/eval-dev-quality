pub fn is_sorted(a: &[i32]) -> bool {
    let mut i = 0;

    while i < a.len() - 1 && a[i] <= a[i + 1] {
        i += 1;
    }

    i == a.len() - 1
}
