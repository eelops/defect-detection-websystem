import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { fetchAlerts } from '../api';

const AlertContext = createContext(null);

function playAlertSound() {
  try {
    const AudioContextClass = window.AudioContext || window.webkitAudioContext;
    if (!AudioContextClass) return;
    const audioContext = new AudioContextClass();
    const oscillator = audioContext.createOscillator();
    const gainNode = audioContext.createGain();

    oscillator.type = 'sawtooth';
    oscillator.frequency.setValueAtTime(880, audioContext.currentTime);
    gainNode.gain.setValueAtTime(0.001, audioContext.currentTime);
    gainNode.gain.exponentialRampToValueAtTime(0.12, audioContext.currentTime + 0.02);
    gainNode.gain.exponentialRampToValueAtTime(0.001, audioContext.currentTime + 0.45);

    oscillator.connect(gainNode);
    gainNode.connect(audioContext.destination);
    oscillator.start();
    oscillator.stop(audioContext.currentTime + 0.45);
  } catch (error) {
    console.error('播放提示音失败', error);
  }
}

function sendDesktopNotification(alert) {
  if (!('Notification' in window) || Notification.permission !== 'granted') {
    return;
  }

  const notification = new Notification('新的严重缺陷告警', {
    body: `${alert.name}｜${alert.defect_type}｜${alert.location || '位置未填写'}`,
    tag: `critical-defect-${alert.id}`,
    requireInteraction: true,
  });

  notification.onclick = () => {
    window.focus();
    window.location.hash = '';
    window.location.href = `/defects/${alert.id}`;
    notification.close();
  };
}

export function AlertProvider({ children }) {
  const [alerts, setAlerts] = useState([]);
  const previousCriticalIdsRef = useRef(new Set());
  const initializedRef = useRef(false);

  const requestNotificationPermission = useCallback(async () => {
    if (!('Notification' in window)) return;
    if (Notification.permission === 'default') {
      try {
        await Notification.requestPermission();
      } catch (error) {
        console.error('请求桌面通知权限失败', error);
      }
    }
  }, []);

  const loadAlerts = useCallback(async () => {
    try {
      const nextAlerts = await fetchAlerts();
      setAlerts(nextAlerts);

      const criticalAlerts = nextAlerts.filter((item) => item.severity === 'critical');
      const nextCriticalIds = new Set(criticalAlerts.map((item) => item.id));

      if (initializedRef.current) {
        const newCriticalAlerts = criticalAlerts.filter((item) => !previousCriticalIdsRef.current.has(item.id));
        if (newCriticalAlerts.length > 0) {
          playAlertSound();
          newCriticalAlerts.forEach(sendDesktopNotification);
        }
      } else {
        initializedRef.current = true;
      }

      previousCriticalIdsRef.current = nextCriticalIds;
    } catch (error) {
      console.error('轮询报警失败', error);
    }
  }, []);

  useEffect(() => {
    requestNotificationPermission();
    loadAlerts();
    const timer = window.setInterval(loadAlerts, 30000);
    return () => window.clearInterval(timer);
  }, [loadAlerts, requestNotificationPermission]);

  const criticalAlerts = useMemo(
    () => alerts.filter((item) => item.severity === 'critical'),
    [alerts]
  );

  const contextValue = useMemo(
    () => ({
      alerts,
      criticalAlerts,
      criticalCount: criticalAlerts.length,
      refreshAlerts: loadAlerts,
    }),
    [alerts, criticalAlerts, loadAlerts]
  );

  return <AlertContext.Provider value={contextValue}>{children}</AlertContext.Provider>;
}

export function useAlerts() {
  const context = useContext(AlertContext);
  if (!context) {
    throw new Error('useAlerts 必须在 AlertProvider 内使用');
  }
  return context;
}
