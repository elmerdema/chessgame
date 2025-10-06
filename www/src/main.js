(async function checkAuthentication() {
    try {
        const response = await fetch('http://localhost:8081/api/check-auth', {
            method: 'GET',
            credentials: 'include' // cookies
        });

        if (!response.ok) {
            // User not authenticated, redirect to the login page
            window.location.replace('auth.html');
            return;
        }
        
        const userData = await response.json();
        
        initializeApp(userData.username);

    } catch (error) {
        window.location.replace('auth.html');
        console.error('Authentication check failed:', error);
    }
})();

function initializeApp(username) {
    console.log(`User '${username}' authenticated. Initializing page elements.`);

    const chatHeader = document.getElementById('chat-header');
    if (chatHeader) {
        chatHeader.textContent = `Chat (${username})`;
    }


}
