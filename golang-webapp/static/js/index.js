document.getElementById('contentForm').addEventListener('submit', function(e) {
    e.preventDefault();
    
    const contentInput = document.getElementById('contentInput');
    const content = contentInput.value.trim();

    if (content) {
        addContentToList(content);
        contentInput.value = '';
    }
});

function addContentToList(content) {
    const contentList = document.getElementById('contentList');
    const li = document.createElement('li');
    li.textContent = content;
    contentList.appendChild(li);
}