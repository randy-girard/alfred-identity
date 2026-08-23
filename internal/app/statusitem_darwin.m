#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>

extern void aiGoShow(void);
extern void aiGoQuit(void);
extern void aiGoCheckUpdates(void);

@interface AIStatusTarget : NSObject
@end

@implementation AIStatusTarget
- (void)onShow:(id)sender { aiGoShow(); }
- (void)onCheckUpdates:(id)sender {
	// Defer until after NSMenu tracking finishes; Wails dialogs crash if shown during menu handling.
	dispatch_async(dispatch_get_main_queue(), ^{
		aiGoCheckUpdates();
	});
}
- (void)onExit:(id)sender { aiGoQuit(); }
- (void)handleReopenEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)reply {
	aiGoShow();
}
@end

static NSStatusItem *gItem;
static AIStatusTarget *gTarget;

static void AIStatusItemCreate(NSString *tooltip, NSData *pngData) {
	if (gItem) {
		return;
	}
	if (!gTarget) {
		gTarget = [AIStatusTarget new];
	}

	[[NSAppleEventManager sharedAppleEventManager]
		setEventHandler:gTarget
		andSelector:@selector(handleReopenEvent:withReplyEvent:)
		forEventClass:kCoreEventClass
		andEventID:kAEReopenApplication];

	NSStatusItem *item = [[NSStatusBar systemStatusBar]
		statusItemWithLength:NSSquareStatusItemLength];
	gItem = item;

	NSStatusBarButton *btn = item.button;
	if (!btn) {
		return;
	}
	btn.toolTip = tooltip ?: @"Alfred Identity";
	btn.target = nil;
	btn.action = NULL;
	btn.enabled = YES;
	btn.appearsDisabled = NO;

	NSImage *image = nil;
	if (pngData.length > 0) {
		image = [[NSImage alloc] initWithData:pngData];
		if (image && image.size.width > 0) {
			image.template = NO;
			image.size = NSMakeSize(18, 18);
		} else {
			image = nil;
		}
	}

	if (image) {
		btn.image = image;
		btn.title = @"";
		btn.imagePosition = NSImageOnly;
	} else {
		btn.image = nil;
		btn.title = @"AI";
		btn.imagePosition = NSNoImage;
	}

	NSMenu *menu = [[NSMenu alloc] initWithTitle:@"Alfred Identity"];
	NSMenuItem *show = [[NSMenuItem alloc] initWithTitle:@"Show Window"
		action:@selector(onShow:) keyEquivalent:@""];
	show.target = gTarget;
	[menu addItem:show];
	NSMenuItem *updates = [[NSMenuItem alloc] initWithTitle:@"Check for Updates…"
		action:@selector(onCheckUpdates:) keyEquivalent:@""];
	updates.target = gTarget;
	[menu addItem:updates];
	[menu addItem:[NSMenuItem separatorItem]];
	NSMenuItem *exitItem = [[NSMenuItem alloc] initWithTitle:@"Exit"
		action:@selector(onExit:) keyEquivalent:@""];
	exitItem.target = gTarget;
	[menu addItem:exitItem];

	item.menu = menu;
	item.visible = YES;
}

void AIStatusItemStart(const char *tooltip, const unsigned char *png, int pngLen) {
	// Copy into ObjC objects *before* returning to Go (Go may free C strings).
	NSString *tip = tooltip ? [NSString stringWithUTF8String:tooltip] : @"Alfred Identity";
	NSData *pngData = nil;
	if (png && pngLen > 0) {
		pngData = [NSData dataWithBytes:png length:(NSUInteger)pngLen];
	}

	void (^create)(void) = ^{
		AIStatusItemCreate(tip, pngData);
	};

	if ([NSThread isMainThread]) {
		create();
	} else {
		// Async: OnStartup runs before the AppKit run loop; sync would deadlock.
		dispatch_async(dispatch_get_main_queue(), create);
	}
}

void AIStatusItemStop(void) {
	void (^stop)(void) = ^{
		[[NSAppleEventManager sharedAppleEventManager]
			removeEventHandlerForEventClass:kCoreEventClass
			andEventID:kAEReopenApplication];
		if (gItem) {
			gItem.menu = nil;
			[[NSStatusBar systemStatusBar] removeStatusItem:gItem];
			gItem = nil;
		}
		gTarget = nil;
	};
	if ([NSThread isMainThread]) {
		stop();
	} else {
		dispatch_async(dispatch_get_main_queue(), stop);
	}
}

void AISetDockVisible(int visible) {
	void (^block)(void) = ^{
		if (visible) {
			[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
			[NSApp activateIgnoringOtherApps:YES];
		} else {
			[NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
		}
	};
	if ([NSThread isMainThread]) {
		block();
	} else {
		dispatch_async(dispatch_get_main_queue(), block);
	}
}

void AIForceShowMainWindow(void) {
	void (^block)(void) = ^{
		[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
		[NSApp activateIgnoringOtherApps:YES];
		for (NSWindow *w in [NSApp windows]) {
			// Skip panels / status-item chrome; deminiaturize on those crashes on newer macOS.
			if (!w.canBecomeKeyWindow && !w.canBecomeMainWindow) {
				continue;
			}
			if ((w.styleMask & NSWindowStyleMaskTitled) == 0) {
				continue;
			}
			@try {
				if (w.miniaturized) {
					[w deminiaturize:nil];
				}
				[w makeKeyAndOrderFront:nil];
			} @catch (NSException *ex) {
				// ignore
			}
		}
	};
	if ([NSThread isMainThread]) {
		block();
	} else {
		dispatch_async(dispatch_get_main_queue(), block);
	}
}
