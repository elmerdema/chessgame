document.addEventListener('DOMContentLoaded', () => {
    //backend URL
    const API_BASE_URL = 'http://localhost:8081';

    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');
    const showRegisterLink = document.getElementById('show-register-link');
    const showLoginLink = document.getElementById('show-login-link');
    const loginErrorEl = document.getElementById('login-error');
    const registerErrorEl = document.getElementById('register-error');

    showRegisterLink.addEventListener('click', (e) => {
        e.preventDefault();
        loginForm.classList.add('hidden');
        registerForm.classList.remove('hidden');
        loginErrorEl.textContent = '';
    });

    showLoginLink.addEventListener('click', (e) => {
        e.preventDefault();
        registerForm.classList.add('hidden');
        loginForm.classList.remove('hidden');
        registerErrorEl.textContent = '';
    });

    loginForm.addEventListener('submit', async (event) => {
        event.preventDefault();
        loginErrorEl.textContent = '';

        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;

        const formData = new URLSearchParams();
        formData.append('username', username);
        formData.append('password', password);

        try {
            // Use the full API URL
            const response = await fetch(`${API_BASE_URL}/login`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
                body: formData,
                // IMPORTANT: This tells the browser to send cookies with the request
                credentials: 'include',
            });

            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText.trim());
            }

            // After login, redirect to the main game page on the frontend server
            window.location.href = '/checkmate.html';

        } catch (error) {
            loginErrorEl.textContent = error.message;
            console.error('Login error:', error);
        }
    });

    registerForm.addEventListener('submit', async (event) => {
        event.preventDefault();
        registerErrorEl.textContent = ''; // Clear previous errors
    
        const username = document.getElementById('register-username').value;
        const password = document.getElementById('register-password').value;
        const confirmPassword = document.getElementById('confirm-password').value;
        
        // --- ADD THIS VALIDATION BLOCK ---
        if (username.length < 5 || password.length < 5) {
            registerErrorEl.textContent = 'Username and password must be at least 5 characters.';
            return; // Stop before sending to the server
        }
    
        if (password !== confirmPassword) {
            registerErrorEl.textContent = 'Passwords do not match.';
            return; // Stop before sending to the server
        }
        // --- END OF VALIDATION BLOCK ---
    
        const formData = new URLSearchParams();
        formData.append('username', username);
        formData.append('password', password);
    
        try {
            // ... the rest of your fetch call remains the same ...
            const response = await fetch(`${API_BASE_URL}/register`, {
                // ...
            });
    
            if (!response.ok) {
                const errorText = await response.text();
                throw new Error(errorText.trim());
            }
    
            const successText = await response.text();
            alert(successText.trim());
    
            registerForm.reset();
            registerForm.classList.add('hidden');
            loginForm.classList.remove('hidden');
    
        } catch (error) {
            registerErrorEl.textContent = error.message;
            console.error('Registration error:', error);
        }
    });
});