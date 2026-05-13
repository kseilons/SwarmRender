package usecase

import "kseilons/worker-service/internall/domain"

func (uc *RenderUseCase) ExportGetWorkerID() string {
	return uc.getWorkerID()
}

func (uc *RenderUseCase) ExportTruncateLogs(s string, maxLen int) string {
	return truncateLogs(s, maxLen)
}

func (uc *RenderUseCase) ExportIsDeviceAllowed(device string) bool {
	return uc.isDeviceAllowed(device)
}

func (uc *RenderUseCase) ExportPrepareJob(task domain.RenderTask, blendPath, outputDir string) domain.RenderJob {
	return uc.prepareJob(task, blendPath, outputDir)
}
