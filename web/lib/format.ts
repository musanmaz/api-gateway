"use client";

import { useEffect, useState } from "react";

const KEY = "gateway.timeZone";
export const ZONES = ["Europe/Istanbul", "UTC", "Europe/London", "America/New_York"];

export function useTimeZone() {
  const [timeZone, setTimeZone] = useState("Europe/Istanbul");
  useEffect(() => {
    const saved = localStorage.getItem(KEY);
    if (saved) setTimeZone(saved);
  }, []);
  function update(next: string) {
    setTimeZone(next);
    localStorage.setItem(KEY, next);
  }
  return { timeZone, setTimeZone: update };
}

export function formatTime(value: string | null | undefined, timeZone: string) {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat("tr-TR", {
    timeZone,
    dateStyle: "medium",
    timeStyle: "medium",
  }).format(date);
}
