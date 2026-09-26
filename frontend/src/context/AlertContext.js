import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { message, notification } from 'antd';
import { fetchAlerts } from '../api';
import { useAuth } from './AuthContext';

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

  const notificationInstance = new Notification('新的严重缺陷告警', {
    body: `${alert.name}｜${alert.defect_type}｜${alert.location || '位置未填写'}`,
    tag: `critical-defect-${alert.id}`,
    requireInteraction: true,
  });

  notificationInstance.onclick = () => {
    window.focus();
    window.location.hash = '';
    window.location.href = `/defects/${alert.id}`;
    notificationInstance.close();
  };
}

function notifyCriticalAlerts(alerts, isImmediate = false) {
  if (!alerts.length) {
    return;
  }
  playAlertSound();
  alerts.forEach((alert) => {
    sendDesktopNotification(alert);
    notification.error({
      message: isImmediate ? '严重缺陷已创建' : '新的严重缺陷告警',
      description: `${alert.name}｜${alert.defect_type}｜${alert.location || '位置未填写'}`,
      duration: 0,
      placement: 'topRight',
    });
  });
  message.error(`当前有 ${alerts.length} 条严重缺陷待处理`);
}

export function AlertProvider({ children }) {
  const { isAuthenticated } = useAuth();
  const [alerts, setAlerts] = useState([]);
  const [alertError, setAlertError] = useState('');
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

  const loadAlerts = useCallback(async (options = {}) => {
    if (!isAuthenticated) {
      setAlerts([]);
      setAlertError('');
      previousCriticalIdsRef.current = new Set();
      initializedRef.current = false;
      return [];
    }
    try {
      const nextAlerts = await fetchAlerts();
      setAlertError('');
      setAlerts(nextAlerts);

      const criticalAlerts = nextAlerts.filter((item) => item.severity === 'critical');
      const nextCriticalIds = new Set(criticalAlerts.map((item) => item.id));

      if (options.immediateNotify) {
        notifyCriticalAlerts(criticalAlerts, true);
      } else if (initializedRef.current) {
        const newCriticalAlerts = criticalAlerts.filter((item) => !previousCriticalIdsRef.current.has(item.id));
        if (newCriticalAlerts.length > 0) {
          notifyCriticalAlerts(newCriticalAlerts);
        }
      } else {
        initializedRef.current = true;
        if (criticalAlerts.length > 0) {
          notifyCriticalAlerts(criticalAlerts);
        }
      }

      previousCriticalIdsRef.current = nextCriticalIds;
      return nextAlerts;
    } catch (error) {
      console.error('轮询报警失败', error);
      setAlertError('告警数据加载失败，请检查后端服务或稍后刷新');
      return [];
    }
  }, [isAuthenticated]);

  useEffect(() => {
    if (!isAuthenticated) {
      setAlerts([]);
      setAlertError('');
      return undefined;
    }
    requestNotificationPermission();
    loadAlerts();
    const timer = window.setInterval(() => loadAlerts(), 30000);
    return () => window.clearInterval(timer);
  }, [isAuthenticated, loadAlerts, requestNotificationPermission]);

  const criticalAlerts = useMemo(
    () => alerts.filter((item) => item.severity === 'critical'),
    [alerts]
  );

  const refreshAlerts = useCallback(
    (options) => loadAlerts(options),
    [loadAlerts]
  );

  const contextValue = useMemo(
    () => ({
      alerts,
      criticalAlerts,
      criticalCount: criticalAlerts.length,
      alertError,
      refreshAlerts,
    }),
    [alerts, criticalAlerts, alertError, refreshAlerts]
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
