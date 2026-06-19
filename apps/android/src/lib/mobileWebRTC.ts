import type { MobilePairingInfo } from "../mobileTypes";
import { parseAgentStreamEvent } from "./agentStream";
import type { AgentStreamEvent } from "./agentStream";

const REQUEST_TIMEOUT_MS = 60_000;
const FRAME_AUTH_CHALLENGE = "auth_challenge";
const FRAME_AUTH_RESPONSE = "auth_response";
const FRAME_AUTH_OK = "auth_ok";
const FRAME_AUTH_ERROR = "auth_error";
const FRAME_REQUEST = "request";
const FRAME_RESPONSE = "response";
const FRAME_STREAM_START = "stream_start";
const FRAME_STREAM_EVENT = "stream_event";
const FRAME_STREAM_END = "stream_end";
const SIGNAL_MOBILE_CONNECT = "mobile.connect";
const SIGNAL_OFFER = "offer";
const SIGNAL_ANSWER = "answer";
const SIGNAL_ICE = "ice";
const SIGNAL_BUSY = "busy";
const SIGNAL_ERROR = "error";
const TURN_URL_PREFIXES = ["turn:", "turns:"] as const;

interface SignalMessage {
  type: string;
  pc_id?: string;
  mobile_id?: string;
  device_id?: string;
  sdp?: string;
  candidate?: RTCIceCandidateInit;
  error?: string;
}

interface DataChannelFrame {
  type: string;
  request_id?: string;
  device_id?: string;
  challenge?: string;
  signature?: string;
  action?: string;
  params?: Record<string, unknown>;
  trace_id?: string;
  status?: "success" | "error";
  payload?: unknown;
  error?: string;
  event?: string;
}

interface PendingRequest {
  resolve: (payload: unknown) => void;
  reject: (error: Error) => void;
  timeout: number;
}

interface PendingStream {
  resolve: (payload: unknown) => void;
  reject: (error: Error) => void;
  onEvent: (event: AgentStreamEvent) => void;
}

export function parsePairingUri(raw: string): { pairing: MobilePairingInfo; secret: string } {
  const parsed = new URL(raw.trim());
  if (parsed.protocol !== "ghost-os:" || parsed.host !== "mobile-pair") {
    throw new Error("配对 URI 格式无效");
  }
  const deviceId = requiredParam(parsed, "device_id");
  const secret = requiredParam(parsed, "secret");
  const pcId = requiredParam(parsed, "pc_id");
  const signalingUrl = requiredParam(parsed, "signaling_url");
  const signalingToken = requiredParam(parsed, "signaling_token");
  const iceServers = parseIceServers(parsed.searchParams.get("ice_servers_json") ?? "");
  return {
    pairing: {
      deviceId,
      pcId,
      signalingUrl,
      signalingToken,
      iceServers,
    },
    secret,
  };
}

export function hasTurnServer(pairing: MobilePairingInfo | undefined): boolean {
  return Boolean(
    pairing?.iceServers.some((server) => {
      const urls = Array.isArray(server.urls) ? server.urls : [server.urls];
      return urls.some(isTurnServerUrl);
    }),
  );
}

export class MobileWebRTCBridge {
  private readonly pairing: MobilePairingInfo;
  private readonly secret: string;
  private readonly pending = new Map<string, PendingRequest>();
  private readonly streams = new Map<string, PendingStream>();
  private channel?: RTCDataChannel;
  private peer?: RTCPeerConnection;
  private pendingRemoteCandidates: RTCIceCandidateInit[] = [];
  private socket?: WebSocket;
  private readyPromise?: Promise<void>;
  private resolveReady?: () => void;
  private rejectReady?: (error: Error) => void;

  constructor(pairing: MobilePairingInfo, secret: string) {
    this.pairing = pairing;
    this.secret = secret;
  }

  async connect(): Promise<void> {
    if (!("RTCPeerConnection" in window)) {
      throw new Error("当前 WebView 不支持 RTCPeerConnection，不能使用 WebRTC 连接");
    }
    this.readyPromise = new Promise((resolve, reject) => {
      this.resolveReady = resolve;
      this.rejectReady = reject;
    });
    this.peer = new RTCPeerConnection({ iceServers: this.pairing.iceServers });
    this.peer.ondatachannel = (event) => this.attachDataChannel(event.channel);
    this.peer.onicecandidate = (event) => {
      if (event.candidate) {
        this.sendSignal({
          type: SIGNAL_ICE,
          pc_id: this.pairing.pcId,
          mobile_id: this.pairing.deviceId,
          device_id: this.pairing.deviceId,
          candidate: event.candidate.toJSON(),
        });
      }
    };
    this.peer.onconnectionstatechange = () => {
      if (this.peer?.connectionState === "failed") {
        this.failAll(this.iceFailureError());
      }
    };

    this.socket = new WebSocket(signalingWebSocketUrl(this.pairing));
    this.socket.onopen = () => {
      this.sendSignal({
        type: SIGNAL_MOBILE_CONNECT,
        pc_id: this.pairing.pcId,
        mobile_id: this.pairing.deviceId,
        device_id: this.pairing.deviceId,
      });
    };
    this.socket.onmessage = (event) => {
      void this.handleSignalMessage(event.data).catch((error: unknown) => {
        this.failAll(errorFromUnknown(error));
      });
    };
    this.socket.onerror = () => {
      this.rejectReady?.(new Error("信令 WebSocket 连接失败"));
    };
    this.socket.onclose = () => {
      this.failAll(new Error("信令 WebSocket 已断开"));
    };
    await this.readyPromise;
  }

  async request<TPayload>(action: string, params: Record<string, unknown>, traceId: string): Promise<TPayload> {
    if (!this.channel || this.channel.readyState !== "open") {
      throw new Error("WebRTC DataChannel 未连接");
    }
    await this.readyPromise;
    const requestId = crypto.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
    const payload = await new Promise<unknown>((resolve, reject) => {
      const timeout = window.setTimeout(() => {
        this.pending.delete(requestId);
        reject(new Error("WebRTC request timed out"));
      }, REQUEST_TIMEOUT_MS);
      this.pending.set(requestId, { resolve, reject, timeout });
      this.sendFrame({
        type: FRAME_REQUEST,
        request_id: requestId,
        action,
        params,
        trace_id: traceId,
      });
    });
    return payload as TPayload;
  }

  async streamAgent<TPayload>(
    params: Record<string, unknown>,
    traceId: string,
    onEvent: (event: AgentStreamEvent) => void,
  ): Promise<TPayload> {
    if (!this.channel || this.channel.readyState !== "open") {
      throw new Error("WebRTC DataChannel 未连接");
    }
    await this.readyPromise;
    const requestId = crypto.randomUUID?.() ?? `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`;
    const payload = await new Promise<unknown>((resolve, reject) => {
      this.streams.set(requestId, { resolve, reject, onEvent });
      this.sendFrame({
        type: FRAME_STREAM_START,
        request_id: requestId,
        action: "AGENT_SEND",
        params,
        trace_id: traceId,
      });
    });
    return payload as TPayload;
  }

  close(): void {
    this.failAll(new Error("WebRTC connection closed"));
    this.channel?.close();
    this.peer?.close();
    this.socket?.close();
  }

  private async handleSignalMessage(raw: unknown): Promise<void> {
    const message = JSON.parse(String(raw)) as SignalMessage;
    switch (message.type) {
      case SIGNAL_OFFER:
        await this.acceptOffer(message);
        break;
      case SIGNAL_ICE:
        if (message.candidate) {
          await this.addRemoteCandidate(message.candidate);
        }
        break;
      case SIGNAL_BUSY:
        this.rejectReady?.(new Error(message.error || "PC 正在被其他设备控制"));
        break;
      case SIGNAL_ERROR:
        this.rejectReady?.(new Error(message.error || "信令服务返回错误"));
        break;
      default:
        break;
    }
  }

  private async acceptOffer(message: SignalMessage): Promise<void> {
    if (!this.peer || !message.sdp) {
      throw new Error("WebRTC offer 缺少 SDP");
    }
    await this.peer.setRemoteDescription({ type: "offer", sdp: message.sdp });
    await this.flushPendingRemoteCandidates();
    const answer = await this.peer.createAnswer();
    await this.peer.setLocalDescription(answer);
    this.sendSignal({
      type: SIGNAL_ANSWER,
      pc_id: this.pairing.pcId,
      mobile_id: this.pairing.deviceId,
      device_id: this.pairing.deviceId,
      sdp: answer.sdp,
    });
  }

  private attachDataChannel(channel: RTCDataChannel): void {
    this.channel = channel;
    channel.onmessage = (event) => {
      void this.handleFrame(event.data);
    };
    channel.onclose = () => this.failAll(new Error("WebRTC DataChannel 已断开"));
    channel.onerror = () => this.failAll(new Error("WebRTC DataChannel 发生错误"));
  }

  private async handleFrame(raw: unknown): Promise<void> {
    const frame = JSON.parse(await frameText(raw)) as DataChannelFrame;
    switch (frame.type) {
      case FRAME_AUTH_CHALLENGE:
        await this.respondToChallenge(frame);
        break;
      case FRAME_AUTH_OK:
        this.resolveReady?.();
        break;
      case FRAME_AUTH_ERROR:
        this.rejectReady?.(new Error(frame.error || "移动设备认证失败"));
        break;
      case FRAME_RESPONSE:
        this.resolveResponse(frame);
        break;
      case FRAME_STREAM_EVENT:
        this.resolveStreamEvent(frame);
        break;
      case FRAME_STREAM_END:
        this.resolveStreamEnd(frame);
        break;
      default:
        break;
    }
  }

  private async respondToChallenge(frame: DataChannelFrame): Promise<void> {
    if (!frame.challenge) {
      throw new Error("认证 challenge 缺失");
    }
    this.sendFrame({
      type: FRAME_AUTH_RESPONSE,
      device_id: this.pairing.deviceId,
      challenge: frame.challenge,
      signature: await signChallenge(this.secret, frame.challenge),
    });
  }

  private resolveResponse(frame: DataChannelFrame): void {
    if (!frame.request_id) {
      return;
    }
    const pending = this.pending.get(frame.request_id);
    if (!pending) {
      return;
    }
    window.clearTimeout(pending.timeout);
    this.pending.delete(frame.request_id);
    if (frame.status === "error") {
      pending.reject(new Error(frame.error || "Bridge returned an error envelope"));
      return;
    }
    pending.resolve(frame.payload);
  }

  private resolveStreamEvent(frame: DataChannelFrame): void {
    if (!frame.request_id) {
      return;
    }
    const pending = this.streams.get(frame.request_id);
    if (!pending) {
      return;
    }

    try {
      pending.onEvent(parseAgentStreamEvent(frame.payload));
    } catch (error) {
      this.streams.delete(frame.request_id);
      pending.reject(errorFromUnknown(error));
    }
  }

  private resolveStreamEnd(frame: DataChannelFrame): void {
    if (!frame.request_id) {
      return;
    }
    const pending = this.streams.get(frame.request_id);
    if (!pending) {
      return;
    }

    this.streams.delete(frame.request_id);
    if (frame.status === "error") {
      pending.reject(new Error(frame.error || "Bridge stream returned an error"));
      return;
    }
    pending.resolve(frame.payload);
  }

  private sendSignal(message: SignalMessage): void {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return;
    }
    this.socket.send(JSON.stringify(message));
  }

  private sendFrame(frame: DataChannelFrame): void {
    if (!this.channel || this.channel.readyState !== "open") {
      return;
    }
    this.channel.send(JSON.stringify(frame));
  }

  private failAll(error: Error): void {
    this.rejectReady?.(error);
    for (const [requestId, pending] of this.pending) {
      window.clearTimeout(pending.timeout);
      pending.reject(error);
      this.pending.delete(requestId);
    }
    for (const [requestId, pending] of this.streams) {
      pending.reject(error);
      this.streams.delete(requestId);
    }
  }

  private async addRemoteCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    if (!this.peer) {
      throw new Error("WebRTC PeerConnection 未初始化");
    }
    if (!this.peer.remoteDescription) {
      this.pendingRemoteCandidates.push(candidate);
      return;
    }
    await this.peer.addIceCandidate(candidate);
  }

  private async flushPendingRemoteCandidates(): Promise<void> {
    if (!this.peer?.remoteDescription) {
      return;
    }
    const candidates = this.pendingRemoteCandidates.splice(0);
    for (const candidate of candidates) {
      await this.peer.addIceCandidate(candidate);
    }
  }

  private iceFailureError(): Error {
    if (hasTurnServer(this.pairing)) {
      return new Error("WebRTC ICE 连接失败");
    }
    return new Error("WebRTC ICE 连接失败：当前配对未包含 TURN，移动网络或公网 NAT 通常无法直连");
  }
}

async function signChallenge(secret: string, challenge: string): Promise<string> {
  if (!crypto.subtle) {
    throw new Error("当前 WebView 不支持 WebCrypto HMAC");
  }
  const key = await crypto.subtle.importKey("raw", base64UrlDecode(secret), { name: "HMAC", hash: "SHA-256" }, false, [
    "sign",
  ]);
  const signature = await crypto.subtle.sign("HMAC", key, new TextEncoder().encode(challenge));
  return base64UrlEncode(new Uint8Array(signature));
}

async function frameText(raw: unknown): Promise<string> {
  if (typeof raw === "string") {
    return raw;
  }
  if (raw instanceof Blob) {
    return raw.text();
  }
  if (raw instanceof ArrayBuffer) {
    return new TextDecoder().decode(raw);
  }
  throw new Error("DataChannel frame must be text JSON");
}

function signalingWebSocketUrl(pairing: MobilePairingInfo): string {
  const url = new URL(pairing.signalingUrl);
  if (url.protocol === "http:") {
    url.protocol = "ws:";
  } else if (url.protocol === "https:") {
    url.protocol = "wss:";
  }
  if (url.protocol !== "ws:" && url.protocol !== "wss:") {
    throw new Error(`信令 URL 协议无效: ${url.protocol}`);
  }
  if (!url.pathname || url.pathname === "/") {
    url.pathname = "/ws";
  }
  url.searchParams.set("token", pairing.signalingToken);
  return url.toString();
}

function parseIceServers(raw: string): RTCIceServer[] {
  if (!raw.trim()) {
    return [];
  }
  const parsed = JSON.parse(raw) as Array<Partial<RTCIceServer> & { URLs?: RTCIceServer["urls"] }>;
  return parsed
    .map((server) => ({
      credential: typeof server.credential === "string" ? server.credential.trim() : server.credential,
      urls: normalizeURLs(server.urls ?? server.URLs),
      username: server.username?.trim(),
    }))
    .filter((server) => (Array.isArray(server.urls) ? server.urls.length > 0 : Boolean(server.urls)));
}

function normalizeURLs(raw: RTCIceServer["urls"] | undefined): string | string[] {
  if (Array.isArray(raw)) {
    return raw.map((value) => value.trim()).filter(Boolean);
  }
  return raw?.trim() ?? "";
}

function isTurnServerUrl(raw: string): boolean {
  const normalized = raw.trim().toLowerCase();
  return TURN_URL_PREFIXES.some((prefix) => normalized.startsWith(prefix));
}

function requiredParam(url: URL, name: string): string {
  const value = url.searchParams.get(name)?.trim();
  if (!value) {
    throw new Error(`配对 URI 缺少 ${name}`);
  }
  return value;
}

function errorFromUnknown(error: unknown): Error {
  if (error instanceof Error) {
    return error;
  }
  return new Error(String(error));
}

function base64UrlDecode(raw: string): Uint8Array {
  const padded = raw.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(raw.length / 4) * 4, "=");
  return Uint8Array.from(atob(padded), (char) => char.charCodeAt(0));
}

function base64UrlEncode(raw: Uint8Array): string {
  let binary = "";
  for (const byte of raw) {
    binary += String.fromCharCode(byte);
  }
  return btoa(binary).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/u, "");
}
