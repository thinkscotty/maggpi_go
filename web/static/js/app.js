// MaggPi Client-Side JavaScript

// Notification system
function showNotification(message, type = 'info') {
    // Remove existing notifications
    const existing = document.querySelector('.notification');
    if (existing) {
        existing.remove();
    }

    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;
    document.body.appendChild(notification);

    // Auto-remove after 5 seconds
    setTimeout(() => {
        notification.style.animation = 'slideIn 0.3s ease reverse';
        setTimeout(() => notification.remove(), 300);
    }, 5000);
}

// Utility function for API calls
async function apiCall(url, options = {}) {
    try {
        const response = await fetch(url, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        });

        const data = await response.json();

        if (!response.ok) {
            throw new Error(data.error || 'Request failed');
        }

        return data;
    } catch (error) {
        console.error('API Error:', error);
        throw error;
    }
}

// Format date for display
function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// Confirm dialog utility
function confirmAction(message) {
    return new Promise((resolve) => {
        resolve(window.confirm(message));
    });
}

// Initialize page-specific functionality
document.addEventListener('DOMContentLoaded', () => {
    // Add loading states to forms
    document.querySelectorAll('form').forEach(form => {
        form.addEventListener('submit', () => {
            const submitBtn = form.querySelector('[type="submit"]');
            if (submitBtn) {
                submitBtn.disabled = true;
                submitBtn.dataset.originalText = submitBtn.textContent;
                submitBtn.textContent = 'Saving...';
            }
        });
    });

    // Handle color input changes for live preview
    const primaryColorInput = document.getElementById('primary-color');
    const secondaryColorInput = document.getElementById('secondary-color');
    const darkModeInput = document.getElementById('dark-mode');

    if (primaryColorInput) {
        primaryColorInput.addEventListener('change', (e) => {
            document.documentElement.style.setProperty('--primary-color', e.target.value);
        });
    }

    if (secondaryColorInput) {
        secondaryColorInput.addEventListener('change', (e) => {
            document.documentElement.style.setProperty('--secondary-color', e.target.value);
        });
    }

    if (darkModeInput) {
        darkModeInput.addEventListener('change', (e) => {
            document.body.classList.toggle('dark-mode', e.target.checked);
        });
    }
});

// Expose functions to global scope for inline handlers
window.showNotification = showNotification;
window.apiCall = apiCall;
window.formatDate = formatDate;
window.confirmAction = confirmAction;
