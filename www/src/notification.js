

let onNotificationCloseCallback = () => {};

// helper functions to get dom elements
function getNotificationElements() {
    const overlay = document.getElementById('ending-overlay');
    const notification = document.getElementById('result-notification');
    const closeBtn = document.getElementById('closeNotification');
    const winnerMessage = document.getElementById('result-message');
    return { overlay, notification, closeBtn, winnerMessage };
}

export function showNotification(winner) {
    const { overlay, notification, closeBtn, winnerMessage } = getNotificationElements();

    if (!overlay || !notification || !winnerMessage) {
        console.error("Notification elements not found for showNotification.");
        return;
    }
    if (winner === "draw") {
        winnerMessage.textContent = "It's a Draw! No Checkmate.";
    }
    else {
        winnerMessage.textContent = `${winner} Wins! Checkmate!`;
    }
    overlay.classList.remove('hidden');
    notification.classList.remove('hidden');

    setTimeout(() => {
        overlay.classList.add('visible');
        notification.classList.add('visible');
        if (closeBtn) closeBtn.focus();
    }, 10);

    overlay.setAttribute('aria-hidden', 'false');
    notification.setAttribute('aria-hidden', 'false');

}

export function hideCheckmateNotification() {
    const { overlay, notification } = getNotificationElements();

    if (!overlay || !notification) {
        console.error("Notification elements not found for hideCheckmateNotification.");
        return;
    }

    overlay.classList.remove('visible');
    notification.classList.remove('visible');

    setTimeout(() => {
        overlay.classList.add('hidden');
        notification.classList.add('hidden');
        if (typeof onNotificationCloseCallback === 'function') {
            onNotificationCloseCallback();
        }
    }, 300);

    overlay.setAttribute('aria-hidden', 'true');
    notification.setAttribute('aria-hidden', 'true');
}

// Function to set the callback from index.js
export function setOnNotificationClose(callback) {
    onNotificationCloseCallback = callback;
}


export function initNotificationEventListeners() {
    const { overlay, closeBtn, notification } = getNotificationElements();

    if (closeBtn) {
        closeBtn.addEventListener('click', hideCheckmateNotification);
    }
    if (overlay) {
        overlay.addEventListener('click', hideCheckmateNotification);
    }
    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && notification && notification.classList.contains('visible')) {
            hideCheckmateNotification();
        }
    });

    // intitial hidden state
    if (notification) notification.setAttribute('aria-hidden', 'true');
    if (overlay) overlay.setAttribute('aria-hidden', 'true');
    if (notification) notification.classList.add('hidden');
    if (overlay) overlay.classList.add('hidden');
}