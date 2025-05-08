pub fn binary_search(a: &[i32], x: i32) -> i32 {
    let mut index = -1;

    let mut min = 0;
    let mut max = a.len() as i32 - 1;

    while index == -1 && min <= max {
        let m = (min + max) / 2;

        if x == a[m as usize] {
            index = m;
        } else if x < a[m as usize] {
            max = m - 1;
        } else {
            min = m + 1;
        }
    }

    index
}
