// app.js
document.addEventListener('DOMContentLoaded', () => {
    const config = getConfig();
    const chatContainer = document.getElementById('chat-container');
    
    chatContainer.innerHTML = `
        <div hx-ext="ws" ws-connect="${config.wsUrl}">
            <div id="chat-messages"></div>
            <form hx-ws="send:submit">
                <input type="text" name="message" placeholder="Type a message...">
                <button type="submit">Send</button>
            </form>
        </div>
    `;
});