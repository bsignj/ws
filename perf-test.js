import { check, sleep } from 'k6';
import ws from 'k6/ws';
import { Trend } from 'k6/metrics';

const url = 'ws://localhost:8383/ws';  // Replace with your WebSocket server URL
const connectTime = new Trend('ws_connect_time');
const sendMessageTime = new Trend('ws_send_message_time');

export let options = {
    stages: [
        { duration: '1m', target: 500 },  // Ramp-up to 100 users over 1 minute
        { duration: '3m', target: 1000 }, // Ramp-up to 1000 users over 3 minutes
        { duration: '2m', target: 1000 }, // Sustain 1000 users for 2 minutes
        { duration: '1m', target: 0 },    // Ramp-down to 0 users
    ],
};

export default function () {
    const vuId = __VU;  // Unique Virtual User ID
    const params = { tags: { vuId: `VU-${vuId}` } };  // Tag WebSocket connection with VU ID

    const res = ws.connect(url, params, function (socket) {
        let startConnectTime = new Date().getTime();

        socket.on('open', function open() {
            let connectTimeTaken = new Date().getTime() - startConnectTime;
            connectTime.add(connectTimeTaken);

            console.log(`[VU-${vuId}] Connected to WebSocket`);

            // Subscribe to the chat room
            socket.send(JSON.stringify({
                type: 'subscribe:chat',
            }));

            // Send a chat message after subscribing
            let startSendMessageTime = new Date().getTime();
            socket.send(JSON.stringify({
                type: 'chat:message',
                payload: {
                    from: `test_user_${vuId}`,  // Unique username for each VU
                    message: `Hello from VU-${vuId}`,
                },
            }));
            let sendMessageTimeTaken = new Date().getTime() - startSendMessageTime;
            sendMessageTime.add(sendMessageTimeTaken);

            // Listen for chat messages
            socket.on('message', function (msg) {
                const messageData = JSON.parse(msg);
                if (messageData.type === 'chat:message') {
                    console.log(`[VU-${vuId}] Received message: ${JSON.stringify(messageData.payload)}`);
                    check(messageData.payload, { 'Message received': (m) => m.message !== '' });
                }
            });

            // Unsubscribe after 5 seconds
            sleep(5);
            socket.send(JSON.stringify({
                type: 'unsubscribe:chat',
            }));

            // Close connection after 10 seconds
            sleep(5);
            socket.close();
        });

        socket.on('close', function close() {
            console.log(`[VU-${vuId}] Disconnected`);
        });

        socket.on('error', function (e) {
            console.error(`[VU-${vuId}] WebSocket error:`, e);
        });
    });

    check(res, { 'Connected successfully': (r) => r && r.status === 101 });
    sleep(1);
}