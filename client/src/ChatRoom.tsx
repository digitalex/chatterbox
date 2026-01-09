import { useState, useRef, useEffect, useLayoutEffect } from 'react';
import { useLiveQuery } from 'dexie-react-hooks';
import { format } from 'date-fns';
import { db } from './db';
import { sendMessage, USER_ID, syncData } from './sync';
import { useAuth } from './AuthContext';
import { RoomOptionsModal } from './RoomOptionsModal';
import {
  BackIcon,
  EditIcon,
  InfoIcon,
  CheckIcon,
  PlusIcon,
  SmileyIcon,
  SendIcon
} from './Icons';

interface ChatRoomProps {
  roomId: string;
  onBack: () => void;
}

export function ChatRoom({ roomId, onBack }: ChatRoomProps) {
    const { user } = useAuth();
    const [isOptionsOpen, setIsOptionsOpen] = useState(false);

    // Pagination state
    const [messageLimit, setMessageLimit] = useState(100);
    const messageListRef = useRef<HTMLDivElement>(null);
    const prevScrollHeightRef = useRef<number>(0);

    const messages = useLiveQuery(async () => {
      // Get the latest N messages (descending order by index, so newest first)
      const msgs = await db.messages
        .where('room_id').equals(roomId)
        .reverse() // Sort by PK descending (so by message_id descending)
        .limit(messageLimit)
        .toArray();
      // Reverse back to chronological order
      return msgs.reverse();
    }, [roomId, messageLimit]);

    const userMap = useLiveQuery(async () => {
      const users = await db.users.toArray();
      return new Map(users.map(u => [u.user_id, u.display_name]));
    });

    const [inputText, setInputText] = useState('');

    const handleSend = async () => {
      if (!inputText.trim()) return;
      const textToSend = inputText;
      setInputText('');
      await sendMessage(roomId, { text: textToSend });
    };

    const roomName = useLiveQuery(
      () => db.rooms.get(roomId),
      [roomId]
    )?.name || 'Chat';

    // Handle scroll to load more
    const handleScroll = () => {
      if (messageListRef.current) {
        if (messageListRef.current.scrollTop === 0) {
          // User reached top, load more
          prevScrollHeightRef.current = messageListRef.current.scrollHeight;
          setMessageLimit((prev) => prev + 50);
        }
      }
    };

    // Scroll to bottom on initial load
    useEffect(() => {
      if (messageListRef.current && messages && messages.length > 0 && messageLimit === 100 && prevScrollHeightRef.current === 0) {
         messageListRef.current.scrollTop = messageListRef.current.scrollHeight;
      }
    }, [messages, messageLimit]);

    // Restore scroll position after loading more
    useLayoutEffect(() => {
        if (messageListRef.current && prevScrollHeightRef.current > 0) {
            const newScrollHeight = messageListRef.current.scrollHeight;
            const diff = newScrollHeight - prevScrollHeightRef.current;
            if (diff > 0) {
                messageListRef.current.scrollTop = diff;
                prevScrollHeightRef.current = 0;
            } else {
               // If diff is 0 or negative, it means no new messages were loaded or layout didn't change (e.g. at end of history).
               // Reset the ref so we don't apply this logic on next unrelated update.
               prevScrollHeightRef.current = 0;
            }
        }
    }, [messages]);

    return (
      <div className="room-view">
        {isOptionsOpen && (
          <RoomOptionsModal
            room={{room_id: roomId, name: roomName}}
            onClose={() => setIsOptionsOpen(false)}
            onUpdate={syncData}
            onDelete={() => {
               syncData();
               onBack(); // Go back to lobby
            }}
          />
        )}
        <div className="chat-header">
          <button className="back-button" onClick={onBack}>
            <BackIcon />
          </button>
          <div className="chat-header-info">
            <div style={{display: 'flex', alignItems: 'center', gap: '8px'}}>
              <span className="chat-title">{roomName}</span>
              {user?.is_admin && (
                <button onClick={() => setIsOptionsOpen(true)} className="icon-btn" style={{padding: 0, border: 'none', background: 'none', cursor: 'pointer', color: '#666'}}>
                  <EditIcon />
                </button>
              )}
            </div>
            <span className="chat-status"><span className="status-dot"></span> Active now</span>
          </div>
          <button className="info-button">
            <InfoIcon />
          </button>
        </div>

        <div
          className="message-list"
          ref={messageListRef}
          onScroll={handleScroll}
        >
          <div className="date-separator"><span>Today</span></div>
          {messages?.map((msg) => {
            const isMe = msg.sender_id === USER_ID;
            return (
              <div key={msg.message_id} className={`message-row ${isMe ? 'outgoing' : 'incoming'}`}>
                {!isMe && <div className="avatar">{userMap?.get(msg.sender_id)?.charAt(0) || '?'}</div>}

                <div className="message-content">
                  {!isMe && <span className="author-name">{userMap?.get(msg.sender_id) || msg.sender_id}</span>}

                  <div className="message-bubble">
                    <div className="body">
                      {(msg.content && typeof msg.content === 'object' && 'text' in msg.content)
                        ? (msg.content as any).text
                        : JSON.stringify(msg.content)
                      }
                    </div>
                  </div>

                  <div className="message-time">
                    {format(new Date(msg.created_at), 'hh:mm a')}
                    {isMe && <span className="read-receipt"><CheckIcon /></span>}
                  </div>
                </div>
              </div>
            );
          })}
        </div>

        <div className="composer-container">
          <button className="composer-btn plus-btn">
             <PlusIcon />
          </button>
          <div className="input-wrapper">
             <input
              type="text"
              placeholder="Type a message..."
              value={inputText}
              onChange={(e) => setInputText(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSend();
              }}
            />
            <button className="smiley-btn"><SmileyIcon /></button>
          </div>
          <button className="send-btn" onClick={handleSend} disabled={!inputText.trim()}>
            <SendIcon />
          </button>
        </div>
      </div>
    );
}
