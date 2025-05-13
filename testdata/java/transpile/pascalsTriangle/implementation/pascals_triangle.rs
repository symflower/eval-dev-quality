pub fn pascals_triangle(rows: i32) -> Result<Vec<Vec<i32>>, String> {
    if rows < 0 {
        return Err("Rows can't be negative!".to_string());
    }

    let rows = rows as usize;
    let mut triangle = Vec::with_capacity(rows);

    for i in 0..rows {
        let mut row = vec![0; i + 1];
        row[0] = 1;

        for j in 1..i {
            row[j] = triangle[i - 1][j - 1] + triangle[i - 1][j];
        }

        if i > 0 {
            row[i] = 1;
        }

        triangle.push(row);
    }

    Ok(triangle)
}
