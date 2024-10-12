import { sleep, check } from 'k6';
import ws from 'k6/ws';
import { Counter } from 'k6/metrics';

export let errorCount = new Counter('errors');

export let options = {
    stages: [
        { duration: '1m', target: 100 },   // Ramp-up to 100 users
        { duration: '3m', target: 100 },   // Stay at 100 users
        { duration: '1m', target: 0 },     // Ramp-down to 0 users
    ],
    thresholds: {
        errors: ['count<10'], // Fail the test if more than 10 errors occur
        http_req_duration: ['p(95)<500'], // 95% of requests must complete below 500ms
    },
};

export default function () {
    const url = 'ws://localhost:8383/ws';
    const params = { tags: { user_id: `${__VU}` } }; // Use VU as user identifier

    const res = ws.connect(url, params, function (socket) {
        socket.on('open', function () {
            console.log('Connected: ' + params.tags.user_id);
            socket.send(JSON.stringify({ type: 'subscribe:chat' }));
        });

        socket.on('message', function (message) {
            console.log('Received message: ' + message);
            check(message, {
                'Message is received': (msg) => msg !== null,
            });
        });

        socket.on('close', function () {
            console.log('Disconnected: ' + params.tags.user_id);
        });

        socket.on('error', function (e) {
            errorCount.add(1);
            console.error('An unexpected error occurred: ', e.error());
        });

        // Send a chat message every 1-3 seconds
        for (let i = 0; i < 10; i++) {
            sleep(Math.random() * 2 + 1);
            socket.send(JSON.stringify({ type: 'chat:message', payload: { from: `User-${__VU}`, message: `Hello from user ${__VU}` } }));
        }

        sleep(1);
        socket.close();
    });

    check(res, { 'Connected successfully': (r) => r && r.status === 101 });
}