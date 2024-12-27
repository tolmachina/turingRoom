// app.js
document.addEventListener('DOMContentLoaded', () => {
    const config = getConfig();
    const chatContainer = document.getElementById('chat-container');
    const role = new URLSearchParams(window.location.search).get('role');
    const roomId = new URLSearchParams(window.location.search).get('room');

    if (role === 'interviewer') {
        setupInterviewerView(chatContainer);
    } else if (role === 'polee') {
        setupPoleeView(chatContainer);
    } else {
        chatContainer.innerHTML = '<p>Invalid role</p>';
        return;
    }

    const socket = new WebSocket(`${config.wsUrl}?role=${role}&room=${roomId}`);

    socket.onopen = () => {
        console.log('WebSocket connection established');
    };

    socket.onmessage = (event) => {
        const message = JSON.parse(event.data);
        displayMessage(message);
    };

    socket.onerror = (error) => {
        console.error('WebSocket error:', error);
    };

    function setupInterviewerView(container) {
        container.innerHTML = `
            <div id="polee-chat" class="chat-window">
                <h2>Chat with Polee</h2>
                <div class="messages"></div>
                <form class="message-form">
                    <input type="text" class="message-input" placeholder="Type a message...">
                    <button type="submit">Send</button>
                </form>
            </div>
            <div id="ai-chat" class="chat-window">
                <h2>Chat with AI</h2>
                <div class="messages"></div>
                <form class="message-form">
                    <input type="text" class="message-input" placeholder="Type a message...">
                    <button type="submit">Send</button>
                </form>
            </div>
        `;
        setupChatHandlers('polee-chat', 'polee');
        setupChatHandlers('ai-chat', 'ai');
    }

    function setupPoleeView(container) {
        container.innerHTML = `
            <div id="interviewer-chat" class="chat-window">
                <h2>Chat with Interviewer</h2>
                <div class="messages"></div>
                <form class="message-form">
                    <input type="text" class="message-input" placeholder="Type a message...">
                    <button type="submit">Send</button>
                </form>
            </div>
        `;
        setupChatHandlers('interviewer-chat', 'interviewer');
    }

    function setupChatHandlers(chatId, recipient) {
        const chatWindow = document.getElementById(chatId);
        const form = chatWindow.querySelector('.message-form');
        const input = chatWindow.querySelector('.message-input');

        form.addEventListener('submit', (e) => {
            e.preventDefault();
            const content = input.value.trim();
            if (content) {
                sendMessage(content, recipient);
                input.value = '';
            }
        });
    }

    function sendMessage(content, to) {
        const message = { content, to };
        socket.send(JSON.stringify(message));
    }

    function displayMessage(message) {
        let chatWindow;
        if (role === 'interviewer') {
            chatWindow = message.from === 'polee' ? document.querySelector('#polee-chat .messages') : document.querySelector('#ai-chat .messages');
        } else {
            chatWindow = document.querySelector('#interviewer-chat .messages');
        }

        const messageElement = document.createElement('div');
        messageElement.textContent = `${message.from}: ${message.content}`;
        chatWindow.appendChild(messageElement);
        chatWindow.scrollTop = chatWindow.scrollHeight;
    }
});