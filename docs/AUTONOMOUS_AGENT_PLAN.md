# Mobile-Optimized Auto-Allow UI Design

## Mobile UX Principles

### 1. Compact Header Indicator (Always Visible)
On mobile, screen real estate is precious. Use a compact indicator in the session header that doubles as a toggle.

```typescript
// MobileHeaderIndicator.tsx
import { Show } from "solid-js"
import { motion } from "solid-motion" // or similar animation library
import { Zap, Bot, Shield } from "lucide-solid"
import { useSessionAutoAllow } from "@/hooks/useSessionAutoAllow"

export function MobileHeaderIndicator() {
  const autoAllow = useSessionAutoAllow()
  
  return (
    <button
      onClick={autoAllow.toggle}
      class={`
        relative flex items-center gap-2 px-3 py-1.5 rounded-full
        transition-all duration-200 active:scale-95
        ${autoAllow.isEnabled() 
          ? 'bg-green-500/20 text-green-600 border border-green-500/30' 
          : 'bg-muted text-muted-foreground border border-transparent'
        }
      `}
    >
      <Show 
        when={autoAllow.isEnabled()} 
        fallback={<Zap class="w-4 h-4" />}
      >
        <motion.div
          animate={{ scale: [1, 1.2, 1] }}
          transition={{ duration: 2, repeat: Infinity }}
        >
          <Bot class="w-4 h-4" />
        </motion.div>
      </Show>
      
      <span class="text-sm font-medium">
        {autoAllow.isEnabled() ? 'AUTO' : 'Manual'}
      </span>
      
      {/* Pulse indicator when active */}
      <Show when={autoAllow.isEnabled()}>
        <span class="absolute top-1 right-1 flex h-2 w-2">
          <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
          <span class="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
        </span>
      </Show>
    </button>
  )
}
```

### 2. Bottom Sheet Settings Panel (Mobile-First)

On mobile, use a bottom sheet instead of a sidebar for settings. This is more thumb-friendly and feels native.

```typescript
// MobileAutoAllowSheet.tsx
import { createSignal, Show } from "solid-js"
import { motion, AnimatePresence } from "solid-motion"
import { 
  Sheet, 
  SheetContent, 
  SheetHeader, 
  SheetTitle, 
  SheetTrigger,
  SheetFooter 
} from "@opencode-ai/ui/sheet"
import { Switch } from "@opencode-ai/ui/switch"
import { Label } from "@opencode-ai/ui/label"
import { Button } from "@opencode-ai/ui/button"
import { Badge } from "@opencode-ai/ui/badge"
import { Separator } from "@opencode-ai/ui/separator"
import { 
  Bot, 
  Zap, 
  ShieldAlert, 
  CheckCircle2, 
  XCircle,
  ChevronRight,
  Settings,
  Activity
} from "lucide-solid"
import { useSessionAutoAllow } from "@/hooks/useSessionAutoAllow"

export function MobileAutoAllowSheet() {
  const autoAllow = useSessionAutoAllow()
  const [isOpen, setIsOpen] = createSignal(false)
  
  const handleToggle = () => {
    autoAllow.toggle()
  }
  
  return (
    <Sheet open={isOpen()} onOpenChange={setIsOpen}>
      <SheetTrigger as={Button} variant="ghost" size="icon">
        <Show 
          when={autoAllow.isEnabled()} 
          fallback={<Settings class="w-5 h-5" />}
        >
          <div class="relative">
            <Zap class="w-5 h-5 text-green-500" />
            <span class="absolute -top-1 -right-1 flex h-2 w-2">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-2 w-2 bg-green-500"></span>
            </span>
          </div>
        </Show>
      </SheetTrigger>
      
      <SheetContent side="bottom" class="h-[85vh] sm:h-auto sm:max-w-lg rounded-t-2xl">
        <SheetHeader class="space-y-4 pb-4">
          <div class="flex items-center justify-between">
            <SheetTitle class="flex items-center gap-2 text-xl">
              <Show 
                when={autoAllow.isEnabled()} 
                fallback={<Bot class="w-6 h-6 text-muted-foreground" />}
              >
                <motion.div
                  animate={{ rotate: [0, 10, -10, 0] }}
                  transition={{ duration: 0.5, repeat: Infinity, repeatDelay: 2 }}
                >
                  <Bot class="w-6 h-6 text-green-500" />
                </motion.div>
              </Show>
              Autonomous Mode
            </SheetTitle>
            
            <Show when={autoAllow.isEnabled()}>
              <Badge class="bg-green-600 hover:bg-green-600 gap-1">
                <CheckCircle2 class="w-3 h-3" />
                ACTIVE
              </Badge>
            </Show>
          </div>
          
          <p class="text-sm text-muted-foreground">
            Control whether this session automatically approves all AI tool permissions.
          </p>
        </SheetHeader>
        
        <div class="space-y-6 py-4">
          {/* Main Toggle */}
          <div class="rounded-xl border p-4 space-y-4">
            <div class="flex items-center justify-between">
              <div class="space-y-1">
                <Label class="text-base font-medium flex items-center gap-2">
                  <Zap class="w-4 h-4" />
                  Auto-allow All Permissions
                </Label>
                <p class="text-xs text-muted-foreground">
                  Automatically approve every tool request
                </p>
              </div>
              <Switch
                checked={autoAllow.isEnabled()}
                onChange={handleToggle}
                class="data-[state=checked]:bg-green-600"
              />
            </div>
            
            <AnimatePresence>
              <Show when={autoAllow.isEnabled()}>
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  class="overflow-hidden"
                >
                  <div class="pt-2 space-y-2">
                    <Separator />
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">Auto-approvals this session</span>
                      <Badge variant="secondary">{autoAllow.approvalCount()}</Badge>
                    </div>
                    <div class="flex items-center justify-between text-sm">
                      <span class="text-muted-foreground">Enabled since</span>
                      <span class="text-xs text-muted-foreground">
                        {autoAllow.enabledAt() 
                          ? new Date(autoAllow.enabledAt()!).toLocaleTimeString() 
                          : '—'}
                      </span>
                    </div>
                  </div>
                </motion.div>
              </Show>
            </AnimatePresence>
          </div>
          
          {/* Security Notice */}
          <div class="rounded-lg bg-amber-500/10 border border-amber-500/20 p-4 space-y-2">
            <div class="flex items-start gap-3">
              <ShieldAlert class="w-5 h-5 text-amber-600 dark:text-amber-400 mt-0.5" />
              <div class="space-y-1">
                <p class="text-sm font-medium text-amber-800 dark:text-amber-200">
                  Security Notice
                </p>
                <p class="text-xs text-amber-700/80 dark:text-amber-300/80">
                  When autonomous mode is enabled, the AI can execute any tool without asking permission. 
                  Only enable this if you fully trust the AI's actions. This setting is saved per session.
                </p>
              </div>
            </div>
          </div>
          
          {/* Quick Actions */}
          <div class="space-y-2">
            <Label class="text-sm font-medium">Quick Actions</Label>
            <div class="grid grid-cols-2 gap-2">
              <Button 
                variant="outline" 
                size="sm"
                class="justify-start gap-2"
                onClick={() => autoAllow.enable()}
                disabled={autoAllow.isEnabled()}
              >
                <CheckCircle2 class="w-4 h-4" />
                Enable Auto-Allow
              </Button>
              <Button 
                variant="outline" 
                size="sm"
                class="justify-start gap-2"
                onClick={() => autoAllow.disable()}
                disabled={!autoAllow.isEnabled()}
              >
                <XCircle class="w-4 h-4" />
                Disable Auto-Allow
              </Button>
            </div>
          </div>
        </div>
        
        <SheetFooter class="pt-4 border-t">
          <div class="flex items-center justify-between w-full text-xs text-muted-foreground">
            <span>Session: {autoAllow.sessionID()?.slice(0, 8)}...</span>
            <span class="flex items-center gap-1">
              <Activity class="w-3 h-3" />
              {autoAllow.isEnabled() ? 'Auto-approving' : 'Manual mode'}
            </span>
          </div>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
```

## Mobile-Optimized Summary

### Key Mobile UI Decisions

1. **Compact Header Indicator**
   - Always visible, one-tap toggle
   - Pulsing animation when active
   - Color-coded (green = active, muted = inactive)

2. **Bottom Sheet Settings**
   - Thumb-friendly bottom sheet (not sidebar)
   - 85vh height for easy reach
   - Large touch targets (44px minimum)
   - Clear visual hierarchy

3. **Touch-Optimized Interactions**
   - Large toggle switches
   - Full-width buttons
   - Swipe to dismiss
   - Haptic feedback (if available)

4. **Visual Feedback**
   - Animated transitions
   - Color-coded status indicators
   - Toast notifications
   - Real-time approval counter

This mobile-first approach ensures the auto-allow feature is easily accessible and usable on all device sizes.