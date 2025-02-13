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

pub fn is_valid_pawn_move(starting_x: i32, starting_y: i32, ending_x: i32, ending_y:i32) -> bool {
    if starting_x == ending_x {
        if starting_y == 1 && ending_y == 3 {
            return true;
        }
        if ending_y == starting_y + 1 {
            return true;
        }
    }
    false

}

pub fn is_valid_rook_move(starting_x: i32, starting_y: i32, ending_x: i32, ending_y: i32) -> bool {
    if starting_x == ending_x || starting_y == ending_y {
        return true;
    }
    false
}

pub fn is_valid_knight_move(starting_x: i32, starting_y: i32, ending_x: i32, ending_y: i32) -> bool {
    if (starting_x - ending_x).abs() == 2 && (starting_y - ending_y).abs() == 1 {
        return true;
    }
    if (starting_x - ending_x).abs() == 1 && (starting_y - ending_y).abs() == 2 {
        return true;
    }
    false
}

pub fn is_valid_bishop_move(starting_x: i32, starting_y: i32, ending_x: i32, ending_y: i32) -> bool {
    if (starting_x - ending_x).abs() == (starting_y - ending_y).abs() {
        return true;
    }
    false
}

pub fn is_valid_queen_move(starting_x: i32, starting_y: i32, ending_x: i32, ending_y: i32) -> bool {
    if is_valid_rook_move(starting_x, starting_y, ending_x, ending_y) || is_valid_bishop_move(starting_x, starting_y, ending_x, ending_y) {
        return true;
    }
    false
}

pub fn is_valid_king_move(starting_x: i32, starting_y: i32, ending_x: i32, ending_y: i32) -> bool {
    if (starting_x - ending_x).abs() <= 1 && (starting_y - ending_y).abs() <= 1 {
        return true;
    }
    false
}
