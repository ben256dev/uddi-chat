const messagesEl = document.getElementById("messages");

let lastEventId = 0;

const ws = new WebSocket(`${location.protocol === "https:" ? "wss:" : "ws:"}//${location.host}/ws`);

ws.addEventListener("open", () => {
	console.log("Connected to UDDI server");
	ws.send(JSON.stringify({ type: "sync", last_event_id: lastEventId }));
});

ws.addEventListener("message", (event) => {
	try {
		const data = JSON.parse(event.data);
		if (data.type === "message") {
			lastEventId = data.event_id;
			console.log("lastEventId:", lastEventId);
			addMessage(data.content);
		}
	} catch (e) {
		// Non-JSON message (e.g. welcome text), ignore
		console.log("Non-JSON message:", event.data);
	}
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
