const messagesEl = document.getElementById("messages");

const ws = new WebSocket(`ws://${location.host}/ws`);

ws.addEventListener("open", () => {
	console.log("Connected to UDDI server");
});

ws.addEventListener("message", (event) => {
	addMessage(event.data);
});

ws.addEventListener("close", () => {
	console.log("Disconnected from UDDI server");
});

function addMessage(text) {
	const msg = document.createElement("div");
	msg.classList.add("message");
	msg.textContent = text;
	messagesEl.appendChild(msg);

	// Remove old messages if there are too many
	while (messagesEl.children.length > 50) {
		messagesEl.removeChild(messagesEl.firstChild);
	}

	// Auto-scroll to bottom
	window.scrollTo(0, document.body.scrollHeight);
}
