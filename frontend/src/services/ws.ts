type WSMessageType = 'job_update' | 'task_update';

interface WSMessage {
  type: WSMessageType;
  job_id?: string;
  task_id?: string;
  status: string;
}

type MessageHandler = (msg: WSMessage) => void;

class WebSocketService {
  private socket: WebSocket | null = null;
  private url: string = 'ws://localhost:8081/ws';
  private handlers: Set<MessageHandler> = new Set();
  private reconnectTimeout: number = 5000;

  constructor() {
    this.connect();
  }

  private connect() {
    console.log('Connecting to WebSocket...');
    this.socket = new WebSocket(this.url);

    this.socket.onopen = () => {
      console.log('WebSocket connected');
    };

    this.socket.onmessage = (event) => {
      try {
        const data: WSMessage = JSON.parse(event.data);
        this.handlers.forEach(handler => handler(data));
      } catch (err) {
        console.error('Failed to parse WS message', err);
      }
    };

    this.socket.onclose = () => {
      console.log('WebSocket closed, reconnecting...');
      setTimeout(() => this.connect(), this.reconnectTimeout);
    };

    this.socket.onerror = (err) => {
      console.error('WebSocket error', err);
      this.socket?.close();
    };
  }

  public subscribe(handler: MessageHandler) {
    this.handlers.add(handler);
    return () => {
      this.handlers.delete(handler);
    };
  }
}

export const wsService = new WebSocketService();
