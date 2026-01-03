// Track Studio Progress Monitor - Next.js/React Example
// This demonstrates how to connect to the SSE endpoint from a Next.js application

import { useEffect, useState } from 'react';

interface ProgressUpdate {
  queue_id: number;
  song_id: number;
  status: string;
  current_step: string;
  progress: number;
  message: string;
  error_message?: string;
  timestamp: string;
}

export function useProgressMonitor() {
  const [updates, setUpdates] = useState<ProgressUpdate[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    // Create EventSource connection to SSE endpoint
    const eventSource = new EventSource('http://localhost:8080/api/v1/progress/stream');

    eventSource.onopen = () => {
      console.log('Connected to progress stream');
      setIsConnected(true);
      setError(null);
    };

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        
        // Skip keepalive messages
        if (data.message === 'connected') {
          console.log('Progress monitor connected:', data);
          return;
        }

        // Add new update to the list
        setUpdates((prev) => [data, ...prev].slice(0, 50)); // Keep last 50 updates
      } catch (err) {
        console.error('Error parsing SSE data:', err);
      }
    };

    eventSource.onerror = (err) => {
      console.error('EventSource error:', err);
      setIsConnected(false);
      setError('Connection lost. Reconnecting...');
      eventSource.close();
    };

    // Cleanup on unmount
    return () => {
      console.log('Closing progress stream connection');
      eventSource.close();
    };
  }, []);

  return { updates, isConnected, error };
}

// Component example
export function ProgressMonitor() {
  const { updates, isConnected, error } = useProgressMonitor();

  return (
    <div className="progress-monitor">
      <div className="status">
        {isConnected ? (
          <span className="connected">🟢 Connected</span>
        ) : (
          <span className="disconnected">🔴 Disconnected</span>
        )}
        {error && <span className="error">{error}</span>}
      </div>

      <div className="updates-list">
        <h3>Live Progress Updates</h3>
        {updates.length === 0 ? (
          <p>No updates yet. Waiting for queue activity...</p>
        ) : (
          updates.map((update, index) => (
            <div key={index} className="update-item">
              <div className="update-header">
                <span className="queue-id">Queue #{update.queue_id}</span>
                <span className="status">{update.status}</span>
                <span className="timestamp">
                  {new Date(update.timestamp).toLocaleTimeString()}
                </span>
              </div>
              <div className="update-body">
                <div className="progress-bar">
                  <div
                    className="progress-fill"
                    style={{ width: `${update.progress}%` }}
                  />
                  <span className="progress-text">{update.progress}%</span>
                </div>
                <div className="details">
                  <strong>Step:</strong> {update.current_step || 'N/A'}
                  <br />
                  <strong>Message:</strong> {update.message}
                </div>
                {update.error_message && (
                  <div className="error-message">{update.error_message}</div>
                )}
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

// Individual Queue Item Monitor
export function useQueueItemMonitor(queueId: number) {
  const [currentProgress, setCurrentProgress] = useState<ProgressUpdate | null>(null);
  const [isConnected, setIsConnected] = useState(false);

  useEffect(() => {
    if (!queueId) return;

    const eventSource = new EventSource(
      `http://localhost:8080/api/v1/progress/stream/${queueId}`
    );

    eventSource.onopen = () => {
      console.log(`Connected to queue ${queueId} progress stream`);
      setIsConnected(true);
    };

    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.queue_id) {
          setCurrentProgress(data);
        }
      } catch (err) {
        console.error('Error parsing SSE data:', err);
      }
    };

    eventSource.onerror = () => {
      setIsConnected(false);
      eventSource.close();
    };

    return () => {
      eventSource.close();
    };
  }, [queueId]);

  return { currentProgress, isConnected };
}

// Queue Item Progress Component
export function QueueItemProgress({ queueId }: { queueId: number }) {
  const { currentProgress, isConnected } = useQueueItemMonitor(queueId);

  if (!currentProgress) {
    return <div>Waiting for progress updates...</div>;
  }

  return (
    <div className="queue-progress">
      <div className="status-badge" data-status={currentProgress.status}>
        {currentProgress.status}
      </div>
      
      <div className="progress-section">
        <h4>{currentProgress.current_step || 'Initializing...'}</h4>
        <div className="progress-bar">
          <div
            className="progress-fill"
            style={{ width: `${currentProgress.progress}%` }}
          >
            {currentProgress.progress}%
          </div>
        </div>
        <p className="message">{currentProgress.message}</p>
      </div>

      {currentProgress.error_message && (
        <div className="error-alert">
          ⚠️ {currentProgress.error_message}
        </div>
      )}

      <div className="connection-status">
        {isConnected ? '🟢 Live' : '🔴 Offline'}
      </div>
    </div>
  );
}
