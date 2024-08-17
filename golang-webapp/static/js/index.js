function addContentToList(content) {
    const contentList = document.getElementById('contentList');
    const li = document.createElement('li');
    li.textContent = content;
    contentList.appendChild(li);
}

function favoriteContent(msgID) {
    fetch('/submitRecommend', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ 'msg_id': msgID }),  // Corrected this line
    })
    .then(response => response.json())
    .then(data => {
        console.log('Success:', data);
    })
    .catch((error) => {
        console.error('Error:', error);
    });
}