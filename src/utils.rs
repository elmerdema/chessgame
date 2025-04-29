pub fn set_panic_hook() {
    // When the `console_error_panic_hook` feature is enabled, we can call the
    // `set_panic_hook` function at least once during initialization, and then
    // we will get better error messages if our code ever panics.
    //
    // For more details see
    // https://github.com/rustwasm/console_error_panic_hook#readme
    #[cfg(feature = "console_error_panic_hook")]
    console_error_panic_hook::set_once();
}

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

/// TODO: This currently only checks moves to empty squares (no captures).
/// TODO: Does not check for obstructions on initial double move.
/// TODO: `is_white` parameter is crucial and must be determined by the caller.
pub fn is_valid_pawn_move(start_x: i32, start_y: i32, end_x: i32, end_y: i32, is_white: bool) -> bool {
    let dx = end_x - start_x;
    let dy = end_y - start_y;

    if is_white { // White pawns move from higher x index to lower x index (e.g., rank 6 to 5 is x=6 to x=5)
        // Move one square forward
        if dx == -1 && dy == 0 {
            return true;
        }
        // Initial two-square move
        if start_x == 6 && dx == -2 && dy == 0 {
             // NOTE: Obstruction check (is board[start_x-1][start_y] empty?) should be done in the main logic
            return true;
        }
        // Captures (currently ignored by lib.rs is_valid_move logic)
        // if dx == -1 && dy.abs() == 1 {
        //     // NOTE: Capture check (is there an opponent piece at end_x, end_y?) should be done in main logic
        //     return true; // Placeholder for capture logic
        // }
    } else { // Black pawns move from lower x index to higher x index (e.g., rank 1 to 2 is x=1 to x=2)
        // Move one square forward
        if dx == 1 && dy == 0 {
            return true;
        }
        // Initial two-square move
        if start_x == 1 && dx == 2 && dy == 0 {
            // NOTE: Obstruction check (is board[start_x+1][start_y] empty?) should be done in the main logic
            return true;
        }
         // Captures (currently ignored by lib.rs is_valid_move logic)
        // if dx == 1 && dy.abs() == 1 {
        //     // NOTE: Capture check (is there an opponent piece at end_x, end_y?) should be done in main logic
        //     return true; // Placeholder for capture logic
        // }
    }

    // Add en passant logic here if needed in the future

    false
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

/// Checks if a bishop move is valid according to its basic movement rules (diagonal).
/// NOTE: Does not check for obstructions along the path.
pub fn is_valid_bishop_move(start_x: i32, start_y: i32, end_x: i32, end_y: i32) -> bool {
    let dx_abs = (end_x - start_x).abs();
    let dy_abs = (end_y - start_y).abs();

    // Move must be purely diagonal (equal change in x and y) and not stationary
    dx_abs == dy_abs && dx_abs != 0
}

/// Checks if a queen move is valid according to its basic movement rules (rook or bishop).
/// NOTE: Does not check for obstructions along the path.
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


// Helper function to check if path is clear (example for Rook horizontal move)
// TODO: Need similar functions for vertical and diagonal paths.
// This function is NOT currently used by lib.rs but would be needed for full validation.
pub fn is_path_clear_horizontal(board: &Vec<Vec<i32>>, start_x: i32, start_y: i32, end_y: i32) -> bool {
    let y_step = if end_y > start_y { 1 } else { -1 };
    let mut current_y = start_y + y_step;
    while current_y != end_y {
        if board[start_x as usize][current_y as usize] != 0 {
            return false; // Path is blocked
        }
        current_y += y_step;
    }
    true // Path is clear
}