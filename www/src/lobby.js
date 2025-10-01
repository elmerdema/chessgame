const API_BASE_URL = "http://localhost:8081/api";
let currentUser = null;

document.addEventListener('DOMContentLoaded', async () => {
    // Check if the user is logged in.
    try {
        const response = await fetch(`${API_BASE_URL}/check-auth`, { credentials: 'include' });
        if (!response.ok) {
            // If the check fails, the user is not logged in.
            // Redirect them to the authentication page.
            window.location.href = '/auth.html';
            return;
        }
        
        const data = await response.json();
        currentUser = data.username;

        document.querySelector('#user-info span').textContent = `Welcome, ${currentUser}!`;

    } catch (error) {

        console.error('Authentication check failed:', error);
        window.location.href = '/auth.html';
        return;
    }

    const findMatchButton = document.getElementById('find-match-button');
    const matchmakingStatus = document.getElementById('matchmaking-status');
    const logoutButton = document.getElementById('logout-button');

    findMatchButton.addEventListener('click', async () => {
        matchmakingStatus.textContent = 'Searching for a match...';
        findMatchButton.disabled = true;
    
        try {
            const response = await fetch(`${API_BASE_URL}/matchmaking/find`, { 
                method: "POST", 
                credentials: 'include'
            });
    
            if (!response.ok) {
                throw new Error(await response.text());
            }
    
            const matchData = await response.json();
    
            // server gives us a gameID, and we go to it.
            window.location.href = `/index.html?gameId=${matchData.gameID}`;
            
        } catch (error) {
            console.error("Error finding match:", error);
            alert("Could not find a match: " + error.message);
            matchmakingStatus.textContent = 'Failed to find a match. Please try again.';
            findMatchButton.disabled = false;
        }
    });

    logoutButton.addEventListener('click', async () => {
        await fetch(`${API_BASE_URL}/logout`, { 
            method: 'POST', 
            credentials: 'include',
            body: new URLSearchParams({ 'username': currentUser })
        });
        
        window.location.href = '/auth.html';
    });
});