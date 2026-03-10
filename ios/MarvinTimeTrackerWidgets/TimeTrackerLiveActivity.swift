import ActivityKit
import SwiftUI
import WidgetKit

struct TimeTrackerLiveActivity: Widget {
    var body: some WidgetConfiguration {
        ActivityConfiguration(for: TimeTrackerAttributes.self) { context in
            // Lock Screen presentation
            lockScreenView(context: context)
        } dynamicIsland: { context in
            DynamicIsland {
                DynamicIslandExpandedRegion(.leading) {
                    Label(context.state.taskTitle, systemImage: "timer")
                        .font(.headline)
                        .lineLimit(1)
                }
                DynamicIslandExpandedRegion(.trailing) {
                    ElapsedTimerText(startedAt: context.state.startedAt, font: .headline)
                }
                DynamicIslandExpandedRegion(.bottom) {
                    Link(destination: .stopTracking) {
                        Label("Stop", systemImage: "stop.fill")
                            .font(.headline)
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .tint(.red)
                }
            } compactLeading: {
                Image(systemName: "timer")
            } compactTrailing: {
                ElapsedTimerText(startedAt: context.state.startedAt)
                    .frame(width: 56)
            } minimal: {
                Image(systemName: "timer")
            }
        }
        .supplementalActivityFamilies([.small])
    }

    @ViewBuilder
    private func lockScreenView(context: ActivityViewContext<TimeTrackerAttributes>) -> some View {
        @Environment(\.activityFamily) var activityFamily

        if activityFamily == .small {
            // Watch Smart Stack layout
            VStack(spacing: 2) {
                Text(context.state.taskTitle)
                    .font(.caption)
                    .lineLimit(1)

                ElapsedTimerText(
                    startedAt: context.state.startedAt,
                    font: .system(size: 28, weight: .medium, design: .monospaced)
                )
            }
            .padding(4)
        } else {
            // iPhone Lock Screen layout
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    Text(context.state.taskTitle)
                        .font(.headline)
                        .lineLimit(1)

                    ElapsedTimerText(
                        startedAt: context.state.startedAt,
                        font: .system(size: 32, weight: .light, design: .monospaced)
                    )
                }

                Spacer()

                Link(destination: .stopTracking) {
                    Image(systemName: "stop.fill")
                        .font(.title2)
                }
                .buttonStyle(.borderedProminent)
                .tint(.red)
            }
            .padding()
        }
    }
}
