export class ChessTimer {
    constructor(gameID) {
        this.gameID = gameID;
        this.whiteTime = 0;
        this.blackTime = 0;
        this.currentTurn = 'white';
        this.isActive = false;
        this.myColor = null;
    }

    setPlayerColor(color) {
        this.myColor = color;
        this.updateDisplay();
    }

    handleTimerUpdate(timerData) {
        this.whiteTime = timerData.whiteTime;
        this.blackTime = timerData.blackTime;
        this.currentTurn = timerData.currentTurn;
        this.isActive = timerData.isActive;

        this.updateDisplay();
    }

    updateDisplay() {
        const myTimerDisplay = document.getElementById('my-timer');
        const opponentTimerDisplay = document.getElementById('opponent-timer');
        const myTimerValue = document.getElementById('my-timer-value');
        const opponentTimerValue = document.getElementById('opponent-timer-value');

        if (!this.myColor) return;

        const myTime = this.myColor === 'white' ? this.whiteTime : this.blackTime;
        const opponentTime = this.myColor === 'white' ? this.blackTime : this.whiteTime;
        const isMyTurn = this.currentTurn.toLowerCase() === this.myColor;

        if (myTimerValue) {
            myTimerValue.textContent = this.formatTime(myTime);
        }
        if (myTimerDisplay) {
            myTimerDisplay.classList.toggle('active', isMyTurn && this.isActive);
            myTimerDisplay.classList.toggle('low-time', myTime < 30);
        }

        if (opponentTimerValue) {
            opponentTimerValue.textContent = this.formatTime(opponentTime);
        }
        if (opponentTimerDisplay) {
            opponentTimerDisplay.classList.toggle('active', !isMyTurn && this.isActive);
            opponentTimerDisplay.classList.toggle('low-time', opponentTime < 30);
        }
    }

    // Format seconds to MM:SS
    formatTime(seconds) {
        if (seconds < 0) seconds = 0;
        const mins = Math.floor(seconds / 60);
        const secs = Math.floor(seconds % 60);
        return `${mins}:${secs.toString().padStart(2, '0')}`;
    }

    initFromGameData(timerData) {
        if (timerData) {
            this.handleTimerUpdate(timerData);
        }
    }
}


// fetch(`/api/game/${gameID}`)
//     .then(res => res.json())
//     .then(data => {
//         if (data.timer) {
//             timer.initFromGameData(data.timer);
//         }
//     });