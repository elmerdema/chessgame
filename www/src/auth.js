document.addEventListener('DOMContentLoaded', () => {
    const loginForm = document.getElementById('login-form');
    const registerForm = document.getElementById('register-form');
    const showRegisterLink = document.getElementById('show-register-link');
    const showLoginLink = document.getElementById('show-login-link');
    const loginError = document.getElementById('login-error');
    const registerError = document.getElementById('register-error');

    showRegisterLink.addEventListener('click', (e) => {
        e.preventDefault();
        loginForm.classList.add('hidden');
        registerForm.classList.remove('hidden');
    });

    showLoginLink.addEventListener('click', (e) => {
        e.preventDefault();
        registerForm.classList.add('hidden');
        loginForm.classList.remove('hidden');
    });

    loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        
        const formData = new URLSearchParams();
        formData.append('username', username);
        formData.append('password', password);
    
        const response = await fetch('http://localhost:8081/login', {
            method: 'POST',
            body: formData
        });
    
        if (response.ok) {
            window.location.href = 'index.html';
        } else {
            const errorMessage = await response.text();
            loginError.textContent = errorMessage;
        }
    });

    registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('register-username').value;
        const password = document.getElementById('register-password').value;
        const confirmPassword = document.getElementById('confirm-password').value;
    
        if (password !== confirmPassword) {
            registerError.textContent = "Passwords do not match!";
            return;
        }
    
        const formData = new URLSearchParams();
        formData.append('username', username);
        formData.append('password', password);
    
        const response = await fetch('http://localhost:8081/register', {
            method: 'POST',
            body: formData
        });
    
        if (response.ok) {
            loginForm.classList.remove('hidden');
            registerForm.classList.add('hidden');
            loginError.textContent = "Registration successful! Please log in.";
            registerError.textContent = '';
        } else {
            const errorMessage = await response.text();
            registerError.textContent = errorMessage;
        }
    });
});
