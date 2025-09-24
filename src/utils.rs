pub fn get_piece_type(piece: i32) ->i32 {
    match piece {
        1..=6 => piece,
        7..=12 => piece -6,
        _ => 0,
    }
}

pub fn get_piece_color(piece: i32) -> i32 {
    match piece {
        1..=6 => 1,
        7..=12 => 2,
        _ => 0,
    }
}

pub fn log(message: &str) {
    println!("{}",message);
}

/// Checks if a rook move is valid according to its basic movement rules (horizontal/vertical).
/// NOTE: Does not check for obstructions along the path.
pub fn is_valid_rook_move(start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
    let dx = end_x - start_x;
    let dy = end_y - start_y;

    // Move must be purely horizontal or purely vertical, and not stationary
    (dx != 0 && dy == 0) || (dx == 0 && dy != 0)
}

/// Checks if a knight move is valid according to its L-shape movement.
pub fn is_valid_knight_move(start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
    let dx_abs = (end_x - start_x).abs();
    let dy_abs = (end_y - start_y).abs();

    // Must move 2 squares in one direction and 1 in the other
    (dx_abs == 2 && dy_abs == 1) || (dx_abs == 1 && dy_abs == 2)
}

pub fn is_valid_bishop_move(start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
    let dx_abs = (end_x - start_x).abs();
    let dy_abs = (end_y - start_y).abs();

    // Move must be purely diagonal (equal change in x and y) and not stationary
    dx_abs == dy_abs && dx_abs != 0
}

/// Checks if a queen move is valid according to its basic movement rules (rook or bishop).
pub fn is_valid_queen_move(start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
    // A queen moves like a rook or a bishop
    is_valid_rook_move(start_x, start_y, end_x, end_y) ||
    is_valid_bishop_move(start_x, start_y, end_x, end_y)
}

/// Checks if a king move is valid according to its basic movement rules (one square any direction).
/// TODO: Does not check for castling.
/// TODO: Does not check if the move puts the king in check (this should be done by the caller).
pub fn is_valid_king_move(start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
    let dx_abs = (end_x - start_x).abs();
    let dy_abs = (end_y - start_y).abs();

    // Can move at most one square horizontally and one square vertically, but must move
    dx_abs <= 1 && dy_abs <= 1 && (dx_abs != 0 || dy_abs != 0)
}
