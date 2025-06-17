// this function runs immediately to guard the page
(async function checkAuthentication() {
    try {
        const response = await fetch('http://localhost:8081/check-auth', {
            method: 'GET',
            credentials: 'include' // cookies
        });

        if (!response.ok) {
            // User not authenticated, redirect to the login page
            window.location.replace('auth.html');
            return;
        }
        
        // User is authenticated, get their data
        const userData = await response.json();

        initializeApp(userData.username);

    } catch (error) {
        window.location.replace('auth.html');
        console.error('Authentication check failed:', error);
    }
})();

// This function only runs for authenticated users
function initializeApp(username) {
    // Wait for the main HTML document to be ready
    document.addEventListener('DOMContentLoaded', () => {
        console.log(`User '${username}' authenticated. Initializing chess app.`);

        // Update UI with username
        const chatHeader = document.getElementById('chat-header');
        if (chatHeader) {
            chatHeader.textContent = `Chat (${username})`;
        }

        //load the WASM bootstrap script
        const bootstrapScript = document.createElement('script');
        bootstrapScript.src = './bootstrap.js';
        document.body.appendChild(bootstrapScript);

        setupWebSocket(username);
    });
}

function setupWebSocket(username) {
    const socket = new WebSocket('ws://localhost:8081/ws');

    socket.onopen = (e) => {
        console.log("WebSocket connection established.");
    };

    socket.onmessage = (e) => {
        const output = document.getElementById('chat-messages');
        const p = document.createElement('p');
        p.textContent = e.data; // The server will now format the message with the sender's name
        output.appendChild(p);
        output.scrollTop = output.scrollHeight;
    };

    socket.onclose = (e) => {
        console.log("WebSocket connection closed.");
    };

    socket.onerror = (e) => {
        console.error("WebSocket error:", e);
    };

    //   attach it to the global window object.
    window.sendMessage = function(event) {
        const messageInput = document.getElementById('chat-input');
        if (event.key === 'Enter' && messageInput.value.trim() !== '') {
            // The server will know who sent it, so we just send the raw message.
            socket.send(messageInput.value);
            messageInput.value = ''; 
        }
    };
}