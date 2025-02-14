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

pub fn get_piece_type(piece: i32) -> i32 {
    // two types of pawns in ourcase, black/white
   let piece_type = match piece {
    1| 6 => 1,
    2| 5 => 2,
    3| 4 => 3,
    7| 12 => 4,
    8| 11 => 5,
    9| 10 => 6,
    _ => 0
    };
    piece_type
   }


pub fn get_piece_value(piece: i32) -> i32 {
    let piece_value = match piece {
        1| 7 => 1,
        2| 8 => 5,
        3| 9 => 3,
        4| 10 => 3,
        5| 11 => 9,
        6| 12 => 1000,
        _ => 0
    };
    piece_value
}


