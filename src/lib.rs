mod utils;

use wasm_bindgen::prelude::*;

#[wasm_bindgen]
struct ChessGame {
    board: Vec<Vec<i32>>,
}

#[wasm_bindgen]
impl ChessGame {
    pub fn new() -> ChessGame {
        let mut board = vec![vec![0;8];8];

        //pieces
        board[0] = vec![1,2,3,4,5,3,2,1];
        board[1] = vec![6,6,6,6,6,6,6,6];
        board[6] = vec![6,6,6,6,6,6,6,6];
        board[7] = vec![1,2,3,4,5,3,2,1];

        ChessGame {
            board
        }
    }

    pub fn get_board(&self) -> Vec<Vec<i32>> {
        self.board.clone()

    }

    pub fn move_piece(&mut self, starting_x: i32, starting_y:i32, ending_x:i32, ending_y:i32) -> bool {
        let piece = self.board[starting_x as usize][starting_y as usize];
        if piece == 0 {
            false
        } 
        else {
            self.board[starting_x as usize][starting_y as usize] = 0;
            self.board[ending_x as usize][ending_y as usize] = piece;
            true
        } 
    }

    pub fn get_piece(&self, x: i32, y: i32) -> i32 {
        self.board[x as usize][y as usize]
    }

     pub fn is_valid_move(&self, starting_x: i32, starting_y: i32, ending_x: i32, ending_y: i32) -> bool { 

        let piece = self.board[starting_x as usize][starting_y as usize];
        if piece == 0 {
            return false;
        }

        let ending_piece = self.board[ending_x as usize][ending_y as usize];
        if ending_piece != 0 {
            return false;
        }

        let piece_type = utils::get_piece_type(piece);
        let is_valid = match piece_type {
            1 => utils::is_valid_pawn_move(starting_x, starting_y, ending_x, ending_y),
            2 => utils::is_valid_rook_move(starting_x, starting_y, ending_x, ending_y),
            3 => utils::is_valid_knight_move(starting_x, starting_y, ending_x, ending_y),
            4 => utils::is_valid_bishop_move(starting_x, starting_y, ending_x, ending_y),
            5 => utils::is_valid_queen_move(starting_x, starting_y, ending_x, ending_y),
            6 => utils::is_valid_king_move(starting_x, starting_y, ending_x, ending_y),
            _ => false
        };

        is_valid
     }

}