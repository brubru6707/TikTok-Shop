function addContentToList(content) {
    const contentList = document.getElementById('contentList');
    const li = document.createElement('li');
    li.textContent = content;
    contentList.appendChild(li);
}