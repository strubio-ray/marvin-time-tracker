import SwiftUI

struct TimerView: View {
    @Bindable var viewModel: TrackingViewModel

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                Spacer()

                switch viewModel.trackingState {
                case .idle:
                    idleView
                case .tracking(_, let title, let startedAt):
                    trackingView(title: title, startedAt: startedAt)
                }

                Spacer()
            }
            .padding()
            .navigationTitle("Marvin Timer")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Menu {
                        Button("Sign Out", role: .destructive) {
                            viewModel.signOut()
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                    }
                }
            }
            .sheet(isPresented: $viewModel.showingTaskPicker) {
                TaskPickerSheet(viewModel: viewModel)
            }
            .task {
                await viewModel.refreshStatus()
            }
        }
    }

    private var idleView: some View {
        VStack(spacing: 16) {
            Image(systemName: "clock")
                .font(.system(size: 48))
                .foregroundStyle(.secondary)

            Text("No task being tracked")
                .font(.headline)
                .foregroundStyle(.secondary)

            Button {
                viewModel.showingTaskPicker = true
            } label: {
                Label("Start Tracking", systemImage: "play.fill")
                    .font(.headline)
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
        }
    }

    private func trackingView(title: String, startedAt: Date) -> some View {
        VStack(spacing: 16) {
            Text(title)
                .font(.title2)
                .fontWeight(.semibold)
                .multilineTextAlignment(.center)

            ElapsedTimerText(
                startedAt: startedAt,
                font: .system(size: 56, weight: .light, design: .monospaced)
            )

            Button(role: .destructive) {
                Task { await viewModel.stopTracking() }
            } label: {
                Label("Stop", systemImage: "stop.fill")
                    .font(.headline)
                    .frame(maxWidth: .infinity)
            }
            .buttonStyle(.borderedProminent)
            .controlSize(.large)
        }
    }
}
