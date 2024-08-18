document.addEventListener('DOMContentLoaded', function () {
    // Add conditional class based on content length
    document.querySelectorAll('#contentList li').forEach(function (item) {
        const contentText = item.querySelector('.content-text').textContent;

        // Assuming "long text" is anything longer than 100 characters
        if (contentText.length > 100) {
            item.classList.add('long-text');
        }
    });

});

function deleteMessage(event, id) {
    event.stopPropagation();  // Prevent the click event from bubbling up to the li element
    console.log(`Delete button clicked for message ID: ${id}`);  // Debugging log

    fetch(`/delete?id=${id}`, {
        method: 'POST'
    })
        .then(response => {
            if (response.ok) {
                // Remove the content item from the DOM
                document.getElementById(`message-${id}`).remove();
            } else {
                console.error("Failed to delete message");
            }
        })
        .catch(error => console.error('Error:', error));
}

function deleteMessage(event, id) {
    event.stopPropagation();  // Prevent the click event from bubbling up to the li element
    console.log(`Delete button clicked for favorite ID: ${id}`);  // Debugging log

    fetch(`/deleteFavorite?id=${id}`, {
        method: 'POST'
    })
        .then(response => {
            if (response.ok) {
                // Remove the content item from the DOM
                document.getElementById(`favorite-${id}`).remove();
            } else {
                console.error("Failed to delete favorite");
            }
        })
        .catch(error => console.error('Error:', error));
}

let socket = new WebSocket("ws://localhost:8080/notifications");

socket.onopen = function () {
    console.log("WebSocket connection established.");
};

socket.onmessage = function (event) {
    console.log("WebSocket message received:", event.data);
    let newMessage = event.data;
    showNotification("New content posted: " + newMessage);
};

socket.onclose = function (event) {
    if (event.wasClean) {
        console.log(`WebSocket connection closed cleanly, code=${event.code} reason=${event.reason}`);
    } else {
        console.error('WebSocket connection closed abruptly.');
    }
};

socket.onerror = function (error) {
    console.error("WebSocket error:", error);
};

window.addEventListener("beforeunload", function () {
    socket.close();
});

// Function to show a notification modal
function showNotification(message) {
    const notificationModal = document.getElementById("notification-modal");
    notificationModal.textContent = message;
    notificationModal.classList.add("show");

    // Hide the notification after 3 seconds
    setTimeout(function () {
        notificationModal.classList.remove("show");
    }, 3000);
}

function favoriteContent(msgID, content) {
    console.log("favoriteContent function called with:", msgID, content);

    const favoriteList = document.getElementById('favoriteList');

    // Create a new list item for the favorite
    const li = document.createElement('li');
    li.id = `favorite-${msgID}`;
    li.innerHTML = `
        ${content}
        <button class="delete" onclick="deleteFavorite(event, '${msgID}')">Remove from Favorites</button>
    `;
    favoriteList.appendChild(li);

    fetch(`/submitRecommend?id=${msgID}`, {
        method: 'POST'
    })
        .then(response => {
            if (response.ok) {
                // Remove the content item from the DOM
                document.getElementById(`message-${id}`).remove();
            } else {
                console.error("Failed to delete message");
            }
        })
        .catch(error => console.error('Error:', error));
}


function deleteFavorite(event, id) {
    event.stopPropagation();  // Prevent the click event from bubbling up to the li element

    if (confirm("Are you sure you want to remove this message from favorites?")) {
        fetch(`/deleteFavorite?id=${id}`, {
            method: 'POST'
        })
            .then(response => {
                if (response.ok) {
                    // Remove the favorite item from the DOM
                    document.getElementById(`favorite-${id}`).remove();
                } else {
                    console.error("Failed to remove favorite message");
                }
            })
            .catch(error => console.error('Error:', error));
    }
}
