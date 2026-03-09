import SwiftUI

@main
struct MarvinTimeTrackerApp: App {
    @UIApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    @State private var viewModel = TrackingViewModel()

    var body: some Scene {
        WindowGroup {
            Group {
                if viewModel.isOnboarded {
                    TimerView(viewModel: viewModel)
                } else {
                    OnboardingView(viewModel: viewModel)
                }
            }
            .task {
                setupAppDelegateCallbacks()
            }
            .task {
                await viewModel.observePushTokens()
            }
            .task {
                await viewModel.observeActivityUpdates()
            }
        }
    }

    private func setupAppDelegateCallbacks() {
        appDelegate.onDeviceTokenRegistered = { token in
            Task {
                await viewModel.registerDeviceToken(token)
            }
        }
        appDelegate.onSilentPushReceived = {
            await viewModel.refreshStatus()
        }
    }
}
