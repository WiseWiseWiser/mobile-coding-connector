import { useRef } from 'react';
import { RemoteScrollbar } from '../components/remote-scrollbar/RemoteScrollbar';
import './ScrollbarMockup.css';

export function ScrollbarMockup() {
    const hContentRef = useRef<HTMLDivElement>(null);
    const vContentRef = useRef<HTMLDivElement>(null);
    const minimalContentRef = useRef<HTMLDivElement>(null);

    return (
        <div className="scrollbar-mockup">
            <div className="sm-header">
                <h2>Remote Scrollbar</h2>
                <p className="sm-description">
                    Decoupled scrollbar that can be placed anywhere to control a target container.
                    Drag the thumb or click on the track to scroll.
                </p>
            </div>

            <div className="sm-section">
                <h3>Horizontal (Scrollbar Below Content)</h3>
                <div className="sm-horizontal-container">
                    <div className="sm-horizontal-content" ref={hContentRef}>
                        {'Lorem ipsum dolor sit amet, consectetur adipiscing elit. '.repeat(20)}
                    </div>
                </div>
                <RemoteScrollbar 
                    targetRef={hContentRef} 
                    orientation="horizontal" 
                    thickness={16}
                />
            </div>

            <div className="sm-section">
                <h3>Vertical (Scrollbar on Right Side)</h3>
                <div className="sm-vertical-layout">
                    <div className="sm-vertical-container">
                        <div className="sm-vertical-content" ref={vContentRef}>
                            {Array.from({ length: 30 }, (_, i) => (
                                <div key={i} className="sm-content-line">
                                    Line {i + 1}: Lorem ipsum dolor sit amet, consectetur adipiscing elit.
                                </div>
                            ))}
                        </div>
                    </div>
                    <RemoteScrollbar 
                        targetRef={vContentRef} 
                        orientation="vertical" 
                        thickness={16}
                    />
                </div>
            </div>

            <div className="sm-section">
                <h3>Minimal (Thumb Only, No Track)</h3>
                <div className="sm-vertical-layout">
                    <div className="sm-vertical-container sm-minimal">
                        <div className="sm-vertical-content" ref={minimalContentRef}>
                            {Array.from({ length: 20 }, (_, i) => (
                                <div key={i} className="sm-content-line">
                                    Minimal scrollbar line {i + 1}
                                </div>
                            ))}
                        </div>
                    </div>
                    <RemoteScrollbar 
                        targetRef={minimalContentRef}
                        orientation="vertical"
                        thickness={8}
                        thumbColor="rgba(239, 68, 68, 0.9)"
                        showTrack={false}
                    />
                </div>
            </div>

            <div className="sm-info">
                <h4>Features:</h4>
                <ul>
                    <li><strong>Decoupled</strong> - Place scrollbar anywhere in UI, control any container</li>
                    <li><strong>iOS Optimized</strong> - Smooth touch handling</li>
                    <li><strong>Drag + Click</strong> - Drag thumb or click track to scroll</li>
                    <li><strong>Auto-sync</strong> - Follows target's scroll position</li>
                    <li><strong>Responsive</strong> - Auto-adjusts to content size</li>
                    <li><strong>Customizable</strong> - Colors, thickness, show/hide track</li>
                </ul>
            </div>
        </div>
    );
}

export default ScrollbarMockup;
