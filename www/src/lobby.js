const API_BASE_URL = "/api";

document.addEventListener('DOMContentLoaded', () => {
    const findMatchButton = document.getElementById('find-match-button');
    const matchmakingStatus = document.getElementById('matchmaking-status');
    const logoutButton = document.getElementById('logout-button');
    const leaderboardBody = document.getElementById('leaderboard-body');
    const loadingOverlay = document.getElementById('loading-overlay'); // New
    const cancelSearchButton = document.getElementById('cancel-search-button'); // New

    let currentUser = null;
    let matchmakingInterval = null;

    const showLoadingPopup = () => {
        loadingOverlay.classList.remove('hidden');
    };

    const hideLoadingPopup = () => {
        loadingOverlay.classList.add('hidden');
    };

    const cancelMatchmaking = () => {
        hideLoadingPopup();
        if (matchmakingInterval) {
            clearInterval(matchmakingInterval);
            matchmakingInterval = null;
        }
        findMatchButton.disabled = false;
        matchmakingStatus.textContent = 'Search canceled. Click the button to find an opponent.';
        // TODO: implement a more robust system,
        // you would send a request to a `/api/matchmaking/cancel` endpoint.
    };

    cancelSearchButton.addEventListener('click', cancelMatchmaking);

    const checkMatchStatus = async () => {
        try {
            const response = await fetch(`${API_BASE_URL}/matchmaking/status`, { credentials: 'include' });
            if (!response.ok) throw new Error('Server returned an error while checking status.');
            const data = await response.json();

            if (data.status === 'found' && data.gameID) {
                console.log('Match found! Game ID:', data.gameID);
                clearInterval(matchmakingInterval);
                matchmakingInterval = null;
                // The popup will disappear when the page redirects
                window.location.href = `/index.html?gameId=${data.gameID}`;
            }
        } catch (error) {
            console.error('Error checking match status:', error);
            alert('An error occurred during matchmaking. Please try again.');
            cancelMatchmaking(); // Use the cancel function to clean up
        }
    };

    findMatchButton.addEventListener('click', async () => {
        if (matchmakingInterval) return;

        findMatchButton.disabled = true;
        showLoadingPopup();

        try {
            const response = await fetch(`${API_BASE_URL}/matchmaking/find`, {
                method: "POST",
                credentials: 'include'
            });

            if (response.status === 202) {
                matchmakingInterval = setInterval(checkMatchStatus, 2500);
            } else if (response.ok) {
                const matchData = await response.json();
                if (matchData.status === 'found' && matchData.gameID) {
                    window.location.href = `/index.html?gameId=${matchData.gameID}`;
                } else {
                    throw new Error('Received an unexpected successful response from server.');
                }
            } else {
                const errorText = await response.text();
                throw new Error(errorText || "Server rejected the match request.");
            }
        } catch (error) {
            console.error("Error starting matchmaking:", error);
            alert("Could not start matchmaking: " + error.message);
            cancelMatchmaking();
        }
    });


    logoutButton.addEventListener('click', async () => {
        if (currentUser) {
            await fetch(`${API_BASE_URL}/logout`, {
                method: 'POST',
                credentials: 'include',
                body: new URLSearchParams({ 'username': currentUser })
            });
        }
        window.location.href = '/auth.html';
    });

    const loadLeaderboard = async () => {
        try {
            const response = await fetch(`${API_BASE_URL}/leaderboard`, { credentials: 'include' });
            if (!response.ok) throw new Error('Failed to fetch leaderboard data');
            const leaderboardData = await response.json();
            leaderboardBody.innerHTML = '';
            leaderboardData.forEach((entry, index) => {
                const row = document.createElement('tr');
                row.innerHTML = `<td>${index + 1}</td><td>${entry.username}</td><td>${entry.elo}</td>`;
                leaderboardBody.appendChild(row);
            });
        } catch (error) {
            console.error('Error loading leaderboard:', error);
            leaderboardBody.innerHTML = '<tr><td colspan="3">Failed to load leaderboard data.</td></tr>';
        }
    };
    
    const init = async () => {
        try {
            const response = await fetch(`${API_BASE_URL}/check-auth`, { credentials: 'include' });
            if (!response.ok) {
                window.location.href = '/auth.html';
                return;
            }
            const data = await response.json();
            currentUser = data.username;
            document.querySelector('#user-info span').textContent = `Welcome, ${currentUser}!`;
            loadLeaderboard();
        } catch (error) {
            console.error('Authentication check failed:', error);
            window.location.href = '/auth.html';
        }
    };

    init();
});