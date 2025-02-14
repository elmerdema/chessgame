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
        board[0]=vec![7, 8, 9, 10, 11, 9, 8, 7];
        board[1]=vec![12, 12, 12, 12, 12, 12, 12, 12];
        board[6]=vec![1, 1, 1, 1, 1, 1, 1, 1];
        board[7]=vec![6, 7, 8, 9, 10, 8, 7, 6];

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

     pub fn is_check(&self, color:i32) -> bool {
        let mut king_x = 0;
        let mut king_y = 0;
        let mut found_king = false;

        for x in 0..8 {
            for y in 0..8 {
                let piece = self.board[x][y];
                if piece == 6 && color == 1 {
                    king_x = x;
                    king_y = y;
                    found_king = true;
                    break;
                }
                if piece == 12 && color == 2 {
                    king_x = x;
                    king_y = y;
                    found_king = true;
                    break;
                }
            }
            if found_king {
                break;
            }
        }

        for x in 0..8 {
            for y in 0..8 {
                let piece = self.board[x][y];
                if piece == 0 {
                    continue;
                }
                if color == 1 && piece > 6 && piece < 13 {
                    continue;
                }
                if color == 2 && piece > 0 && piece < 7 {
                    continue;
                }

                if self.is_valid_move(x as i32, y as i32, king_x as i32, king_y as i32) {
                    return true;
                }
            }
        }
        false
     }

     

}