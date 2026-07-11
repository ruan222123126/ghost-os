import { describe, expect, it } from "vitest";
import type { MobilePairingInfo } from "../mobileTypes";
import { hasTurnServer } from "./mobileWebRTC";

describe("mobile WebRTC pairing", () => {
  it("detects TURN URLs in pairing ICE servers", () => {
    expect(hasTurnServer(pairingWith([{ urls: "turn:turn.example.com:3478" }]))).toBe(true);
    expect(hasTurnServer(pairingWith([{ urls: ["stun:stun.example.com:3478", "turns:turn.example.com:5349"] }]))).toBe(
      true,
    );
  });

  it("does not report TURN for STUN-only pairings", () => {
    expect(hasTurnServer(pairingWith([{ urls: "stun:stun.example.com:3478" }]))).toBe(false);
    expect(hasTurnServer(undefined)).toBe(false);
  });
});

function pairingWith(iceServers: RTCIceServer[]): MobilePairingInfo {
  return {
    deviceId: "device-1",
    iceServers,
    pcId: "pc-1",
    signalingToken: "token",
    signalingUrl: "wss://signaling.example.com/ws",
  };
}
