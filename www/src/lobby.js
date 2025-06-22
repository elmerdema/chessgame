document.addEventListener('DOMContentLoaded', () => {
    const findMatchButton = document.getElementById('find-match-button');
    const matchmakingStatus = document.getElementById('matchmaking-status');
    const logoutButton = document.getElementById('logout-button');

    findMatchButton.addEventListener('click', () => {
        matchmakingStatus.textContent = 'Searching for a match...';
        // In a real application, this would involve a call to a matchmaking service.
        // For now, we'll just simulate finding a match and redirecting.
        setTimeout(() => {
            window.location.href = 'index.html';
        }, 3000); // Simulate a 3-second search
    });

    logoutButton.addEventListener('click', () => {
        // Handle logout logic, e.g., redirecting to a login page
        console.log('Logout clicked');
        // window.location.href = 'login.html';
    });

    // Fetch and display leaderboard data
    // This is a placeholder. In a real app, you would fetch this from a server.
    const leaderboardBody = document.getElementById('leaderboard-body');
    // You can clear existing rows if you're populating dynamically
    // leaderboardBody.innerHTML = ''; 
}); 