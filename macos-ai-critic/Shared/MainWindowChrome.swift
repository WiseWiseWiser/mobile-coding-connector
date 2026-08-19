import SwiftUI

/// NavigationSplitView shell: sidebar + injected page for the selected item.
@available(macOS 15.0, *)
public struct MainWindowChrome<Page: View>: View {
    @ObservedObject private var router = MainWindowRouter.shared
    private let page: (MainSidebarItem) -> Page

    public init(@ViewBuilder page: @escaping (MainSidebarItem) -> Page) {
        self.page = page
    }

    public var body: some View {
        NavigationSplitView {
            List(MainSidebarItem.allCases, selection: Binding<MainSidebarItem?>(
                get: { router.selection },
                set: { router.selection = $0 ?? .home }
            )) { item in
                Label(item.title, systemImage: item.systemImage)
                    .tag(item)
            }
            .listStyle(.sidebar)
            .navigationSplitViewColumnWidth(min: 140, ideal: 160, max: 200)
        } detail: {
            page(router.selection)
        }
        .frame(minWidth: 720, minHeight: 520)
        .accessibilityIdentifier("main-window")
    }
}
