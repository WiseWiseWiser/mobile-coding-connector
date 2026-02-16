import { useState } from 'react';
import { NoZoomingInput } from '../v2/mcc/components/NoZoomingInput';
import { MockupPageContainer } from './MockupPageContainer';
import './NoZoomingInputDemo.css';

export function NoZoomingInputDemo() {
    const [basicInput, setBasicInput] = useState('');
    const [basicTextarea, setBasicTextarea] = useState('');
    const [inputWithButton, setInputWithButton] = useState('');
    const [chatMessages, setChatMessages] = useState<string[]>([]);
    const [chatInput, setChatInput] = useState('');

    const handleSendChat = () => {
        if (!chatInput.trim()) return;
        setChatMessages([...chatMessages, chatInput.trim()]);
        setChatInput('');
    };

    return (
        <MockupPageContainer 
            title="No Zooming Input Demo"
            description="These inputs use font-size: 16px and touch-action: manipulation to prevent iOS Safari from zooming when focused."
        >
            {/* Basic Input */}
            <div className="nozoom-demo-section">
                <h3>1. Basic Input</h3>
                <NoZoomingInput>
                    <input
                        type="text"
                        className="nozoom-demo-input"
                        placeholder="Type here..."
                        value={basicInput}
                        onChange={e => setBasicInput(e.target.value)}
                    />
                </NoZoomingInput>
            </div>

            {/* Basic Textarea */}
            <div className="nozoom-demo-section">
                <h3>2. Basic Textarea</h3>
                <NoZoomingInput>
                    <textarea
                        className="nozoom-demo-textarea"
                        placeholder="Enter your message..."
                        value={basicTextarea}
                        onChange={e => setBasicTextarea(e.target.value)}
                        rows={4}
                    />
                </NoZoomingInput>
            </div>

            {/* Input with Send Button */}
            <div className="nozoom-demo-section">
                <h3>3. Input with Send Button</h3>
                <div className="nozoom-demo-row">
                    <NoZoomingInput style={{ flex: 1 }}>
                        <input
                            type="text"
                            className="nozoom-demo-input"
                            placeholder="Type and send..."
                            value={inputWithButton}
                            onChange={e => setInputWithButton(e.target.value)}
                        />
                    </NoZoomingInput>
                    <button
                        className="nozoom-demo-btn"
                        onClick={() => setInputWithButton('')}
                    >
                        Send
                    </button>
                </div>
            </div>

            {/* Chat-like Interface */}
            <div className="nozoom-demo-section">
                <h3>4. Chat Interface</h3>
                <div className="nozoom-demo-chat">
                    <div className="nozoom-demo-chat-messages">
                        {chatMessages.length === 0 ? (
                            <div className="nozoom-demo-chat-empty">No messages yet</div>
                        ) : (
                            chatMessages.map((msg, i) => (
                                <div key={i} className="nozoom-demo-chat-msg">{msg}</div>
                            ))
                        )}
                    </div>
                    <div className="nozoom-demo-chat-input-row">
                        <NoZoomingInput style={{ flex: 1 }}>
                            <textarea
                                className="nozoom-demo-chat-input"
                                placeholder="Type a message..."
                                value={chatInput}
                                onChange={e => setChatInput(e.target.value)}
                                rows={2}
                            />
                        </NoZoomingInput>
                        <button
                            className="nozoom-demo-btn"
                            onClick={handleSendChat}
                        >
                            Send
                        </button>
                    </div>
                </div>
            </div>
        </MockupPageContainer>
    );
}
