let socket = new WebSocket("ws://localhost:8080/ws");
console.log("Attempting Connection...");

socket.onopen = () => {
    console.log("Successfully Connected");
    socket.send("Hi From the Client!");
}

socket.onclose = event => {
    console.log("Socket Connection Closed: ", event);
    socket.send("Client Closed!");
}

socket.onerror = error => {
    console.log("Socket Error: ", error);
    socket.send("Client Error!");
}
