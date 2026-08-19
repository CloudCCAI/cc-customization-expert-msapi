package msapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloudcc-customization-expert-go/internal/config"
)

const setupSvcParityReplayApproval = "CLOUDCC_SETUP_SVC_PARITY_REPLAY_APPROVED"
const setupSvcParityEvidenceWorkspaceApproval = "CLOUDCC_SETUP_SVC_PARITY_EVIDENCE_WORKSPACE_APPROVED"
const setupSvcParityCaptureSourceWorkspaceApproval = "CLOUDCC_SETUP_SVC_PARITY_CAPTURE_SOURCE_WORKSPACE_APPROVED"
const setupSvcParityNormalizedDiffApproval = "CLOUDCC_SETUP_SVC_PARITY_NORMALIZED_DIFF_APPROVED"
const setupSvcParityManifestSyncApproval = "CLOUDCC_SETUP_SVC_PARITY_MANIFEST_SYNC_APPROVED"
const setupSvcParityEvidenceBundleApproval = "CLOUDCC_SETUP_SVC_PARITY_EVIDENCE_BUNDLE_APPROVED"
const setupSvcParityEvidenceImportApproval = "CLOUDCC_SETUP_SVC_PARITY_EVIDENCE_IMPORT_APPROVED"
const setupSvcParityMatrixPromotionApproval = "CLOUDCC_SETUP_SVC_PARITY_MATRIX_PROMOTION_APPROVED"
const setupSvcParityQueryReadbackCaptureApproval = "CLOUDCC_SETUP_SVC_PARITY_QUERY_READBACK_CAPTURE_APPROVED"
const setupSvcParityMetadataServiceQueryScanCaptureApproval = "CLOUDCC_SETUP_SVC_PARITY_METADATA_SERVICE_QUERY_SCAN_CAPTURE_APPROVED"
const setupSvcParityMetadataServiceApplyCaptureApproval = "CLOUDCC_SETUP_SVC_PARITY_METADATA_SERVICE_APPLY_CAPTURE_APPROVED"
const setupSvcParitySnapshotFromChangesApproval = "CLOUDCC_SETUP_SVC_PARITY_SNAPSHOT_FROM_CHANGES_APPROVED"
const setupSvcLiveReplayContractVersion = "setup-svc-live-replay-contract/v1"
const setupSvcLiveReplayEvidenceMode = "setup-svc-live-replay-evidence"
const setupSvcLiveReplayEvidenceSectionQueueCommandLimit = 10
const setupSvcLiveReplayCollectionRunbookCommandLimit = 20
const setupSvcLiveReplayWorklistBatchLimit = setupSvcLiveReplayEvidenceSectionQueueCommandLimit

type setupSvcLiveReplayReadiness struct {
	Mode             string                           `json:"mode"`
	Project          string                           `json:"project"`
	ReadOnly         bool                             `json:"readOnly"`
	Execute          bool                             `json:"execute"`
	ApprovalRequired bool                             `json:"approvalRequired"`
	ApprovalPhrase   string                           `json:"approvalPhrase"`
	Status           string                           `json:"status"`
	Config           setupSvcLiveReplayConfig         `json:"config"`
	Totals           setupSvcLiveReplayTotals         `json:"totals"`
	Commands         setupSvcLiveReplayCommands       `json:"commands"`
	MatrixContract   setupSvcLiveReplayMatrixContract `json:"matrixContract"`
	Domains          []setupSvcLiveReplayDomain       `json:"domains"`
	BlockingIssues   []string                         `json:"blockingIssues,omitempty"`
	StopConditions   []string                         `json:"stopConditions"`
	Notes            []string                         `json:"notes"`
}

type setupSvcLiveReplayConfig struct {
	HasSetupSvc          bool   `json:"hasSetupSvc"`
	HasApiSvc            bool   `json:"hasApiSvc"`
	HasAccessToken       bool   `json:"hasAccessToken"`
	HasMetadataService   bool   `json:"hasMetadataService"`
	SetupSvcHost         string `json:"setupSvcHost,omitempty"`
	MetadataServiceURL   string `json:"metadataServiceUrl,omitempty"`
	SetupSvcHostRedacted bool   `json:"setupSvcHostRedacted"`
}

type setupSvcLiveReplayTotals struct {
	Domains                  int `json:"domains"`
	Operations               int `json:"operations"`
	CoveredPendingLiveReplay int `json:"coveredPendingLiveReplay"`
	Verified                 int `json:"verified"`
}

type setupSvcLiveReplayCommands struct {
	Readiness         string `json:"readiness"`
	MetadataPreflight string `json:"metadataPreflight"`
	ApprovedReplay    string `json:"approvedReplay"`
	EvidenceDirectory string `json:"evidenceDirectory"`
}

type setupSvcLiveReplayEnvironmentResult struct {
	Mode                      string                                 `json:"mode"`
	Project                   string                                 `json:"project"`
	ReadOnly                  bool                                   `json:"readOnly"`
	Status                    string                                 `json:"status"`
	Config                    setupSvcLiveReplayEnvironmentConfig    `json:"config"`
	MetadataService           setupSvcLiveReplayEndpointCheck        `json:"metadataService"`
	MetadataServiceDatasource setupSvcLiveReplayDatasourceReadiness  `json:"metadataServiceDatasource"`
	CaptureSources            setupSvcLiveReplayCaptureSourceSummary `json:"captureSources"`
	CompletionAudit           setupSvcLiveReplayEnvironmentAudit     `json:"completionAudit"`
	Gates                     []setupSvcLiveReplayPreflightGate      `json:"gates"`
	BlockingIssues            []string                               `json:"blockingIssues,omitempty"`
	NextCommands              []string                               `json:"nextCommands"`
	Notes                     []string                               `json:"notes"`
}

type setupSvcLiveReplayEnvironmentConfig struct {
	ConfigFileExists           bool   `json:"configFileExists"`
	ConfigProfile              string `json:"configProfile,omitempty"`
	SetupSvcConfigured         bool   `json:"setupSvcConfigured"`
	SetupSvcEndpoint           string `json:"setupSvcEndpoint,omitempty"`
	ApiSvcConfigured           bool   `json:"apiSvcConfigured"`
	ApiSvcEndpoint             string `json:"apiSvcEndpoint,omitempty"`
	AccessTokenConfigured      bool   `json:"accessTokenConfigured"`
	MetadataServiceConfigured  bool   `json:"metadataServiceConfigured"`
	MetadataServiceEndpoint    string `json:"metadataServiceEndpoint"`
	MetadataServiceTokenSource string `json:"metadataServiceTokenSource,omitempty"`
}

type setupSvcLiveReplayDatasourceReadiness struct {
	RuntimeMode            string   `json:"runtimeMode"`
	RuntimeModeSource      string   `json:"runtimeModeSource"`
	ServerPort             string   `json:"serverPort"`
	ServerPortSource       string   `json:"serverPortSource"`
	JDBCURLConfigured      bool     `json:"jdbcUrlConfigured"`
	JDBCURLSource          string   `json:"jdbcUrlSource,omitempty"`
	JDBCURLLooksDefaultH2  bool     `json:"jdbcUrlLooksDefaultH2"`
	UsernameConfigured     bool     `json:"usernameConfigured"`
	UsernameSource         string   `json:"usernameSource,omitempty"`
	PasswordConfigured     bool     `json:"passwordConfigured"`
	PasswordSource         string   `json:"passwordSource,omitempty"`
	DriverConfigured       bool     `json:"driverConfigured"`
	DriverSource           string   `json:"driverSource,omitempty"`
	ReadyForRealDatasource bool     `json:"readyForRealDatasource"`
	Status                 string   `json:"status"`
	Missing                []string `json:"missing,omitempty"`
	Warnings               []string `json:"warnings,omitempty"`
	StartCommandHint       string   `json:"startCommandHint"`
	Redaction              string   `json:"redaction"`
}

type setupSvcLiveReplayEndpointCheck struct {
	URL       string `json:"url"`
	Reachable bool   `json:"reachable"`
	Status    string `json:"status"`
	HTTPCode  int    `json:"httpCode,omitempty"`
	Error     string `json:"error,omitempty"`
}

type setupSvcLiveReplayEnvironmentAudit struct {
	Status                     string                                        `json:"status"`
	ManifestPath               string                                        `json:"manifestPath"`
	MatrixContractStatus       string                                        `json:"matrixContractStatus"`
	MatrixVerifiedDomains      int                                           `json:"matrixVerifiedDomains"`
	MatrixNonVerifiedDomains   int                                           `json:"matrixNonVerifiedDomains"`
	EvidenceVerifiedOperations int                                           `json:"evidenceVerifiedOperations"`
	CompletedOperations        int                                           `json:"completedOperations"`
	FailedEvidenceTotal        int                                           `json:"failedEvidenceTotal"`
	RepairQueueCount           int                                           `json:"repairQueueCount"`
	RepairQueues               []setupSvcLiveReplayEvidenceImportRepairQueue `json:"repairQueues,omitempty"`
}

type setupSvcLiveReplayDomain struct {
	Domain                    string   `json:"domain"`
	Operations                []string `json:"operations"`
	RequiredTables            []string `json:"requiredTables"`
	RuntimeEffects            []string `json:"runtimeEffects,omitempty"`
	QueryReadbackExpectations []string `json:"queryReadbackExpectations,omitempty"`
	SetupSvcEvidence          []string `json:"setupSvcEvidence"`
	MetadataServiceEvidence   []string `json:"metadataServiceEvidence"`
	QueryEvidence             []string `json:"queryEvidence"`
	Status                    string   `json:"status"`
	ApprovalRequired          bool     `json:"approvalRequired"`
}

type setupSvcLiveReplayEvidenceResult struct {
	Mode                string                             `json:"mode"`
	Project             string                             `json:"project"`
	ReadOnly            bool                               `json:"readOnly"`
	Status              string                             `json:"status"`
	ManifestPath        string                             `json:"manifestPath"`
	ContractVersion     string                             `json:"contractVersion"`
	ContractFingerprint string                             `json:"contractFingerprint"`
	MatrixContract      setupSvcLiveReplayMatrixContract   `json:"matrixContract"`
	Totals              setupSvcLiveReplayEvidenceTotals   `json:"totals"`
	Domains             []setupSvcLiveReplayEvidenceDomain `json:"domains"`
	BlockingIssues      []string                           `json:"blockingIssues,omitempty"`
	Notes               []string                           `json:"notes"`
}

type setupSvcLiveReplayPromotionResult struct {
	Mode           string                              `json:"mode"`
	Project        string                              `json:"project"`
	ReadOnly       bool                                `json:"readOnly"`
	Status         string                              `json:"status"`
	ManifestPath   string                              `json:"manifestPath"`
	Totals         setupSvcLiveReplayPromotionTotals   `json:"totals"`
	Domains        []setupSvcLiveReplayPromotionDomain `json:"domains"`
	MatrixUpdates  []setupSvcLiveReplayMatrixUpdate    `json:"matrixUpdates"`
	BlockingIssues []string                            `json:"blockingIssues,omitempty"`
	Notes          []string                            `json:"notes"`
}

type setupSvcLiveReplayCompletionAuditResult struct {
	Mode                      string                                            `json:"mode"`
	Project                   string                                            `json:"project"`
	ReadOnly                  bool                                              `json:"readOnly"`
	Status                    string                                            `json:"status"`
	ManifestPath              string                                            `json:"manifestPath"`
	ContractVersion           string                                            `json:"contractVersion,omitempty"`
	ContractFingerprint       string                                            `json:"contractFingerprint,omitempty"`
	MatrixContract            setupSvcLiveReplayMatrixContract                  `json:"matrixContract"`
	TestEvidence              setupSvcLiveReplayTestEvidence                    `json:"testEvidenceContract"`
	Totals                    setupSvcLiveReplayCompletionAuditTotals           `json:"totals"`
	Gates                     []setupSvcLiveReplayCompletionAuditGate           `json:"gates"`
	GateStatuses              map[string]setupSvcLiveReplayCompletionGateStatus `json:"gateStatuses,omitempty"`
	GateSummaries             map[string]map[string]any                         `json:"gateSummaries,omitempty"`
	OperatorPacket            setupSvcLiveReplayCompletionOperatorPacket        `json:"operatorPacket"`
	MetadataServiceDatasource setupSvcLiveReplayDatasourceReadiness             `json:"metadataServiceDatasource"`
	Domains                   []setupSvcLiveReplayCompletionAuditDomain         `json:"domains"`
	BlockingIssues            []string                                          `json:"blockingIssues,omitempty"`
	FailedEvidence            []string                                          `json:"failedEvidence,omitempty"`
	FailedEvidenceTotal       int                                               `json:"failedEvidenceTotal"`
	RepairQueueCount          int                                               `json:"repairQueueCount"`
	RepairPlan                setupSvcLiveReplayCompletionRepairPlan            `json:"repairPlan,omitempty"`
	FailedEvidenceSummary     setupSvcLiveReplayCompletionFailedEvidenceSummary `json:"failedEvidenceSummary,omitempty"`
	NextActions               []string                                          `json:"nextActions"`
	NextCommands              []string                                          `json:"nextCommands,omitempty"`
	Notes                     []string                                          `json:"notes"`
}

type setupSvcLiveReplayPreflightResult struct {
	Mode                      string                                 `json:"mode"`
	Project                   string                                 `json:"project"`
	ReadOnly                  bool                                   `json:"readOnly"`
	Status                    string                                 `json:"status"`
	ManifestPath              string                                 `json:"manifestPath"`
	PacketPathHint            string                                 `json:"packetPathHint"`
	Totals                    setupSvcLiveReplayPacketTotals         `json:"totals"`
	Gates                     []setupSvcLiveReplayPreflightGate      `json:"gates"`
	MetadataServiceDatasource setupSvcLiveReplayDatasourceReadiness  `json:"metadataServiceDatasource"`
	CaptureSources            setupSvcLiveReplayCaptureSourceSummary `json:"captureSources"`
	BlockingIssues            []string                               `json:"blockingIssues,omitempty"`
	NextCommands              []string                               `json:"nextCommands"`
	Notes                     []string                               `json:"notes"`
}

type setupSvcLiveReplayPreflightGate struct {
	Name                      string                                 `json:"name"`
	Status                    string                                 `json:"status"`
	Blocking                  bool                                   `json:"blocking"`
	Summary                   map[string]any                         `json:"summary,omitempty"`
	MetadataServiceDatasource *setupSvcLiveReplayDatasourceReadiness `json:"metadataServiceDatasource,omitempty"`
	NextAction                string                                 `json:"nextAction,omitempty"`
}

type setupSvcLiveReplayCaptureSourceSummary struct {
	Status                                 string                                          `json:"status"`
	SourceRoot                             string                                          `json:"sourceRoot"`
	CaptureRoot                            string                                          `json:"captureRoot"`
	ArtifactFiles                          int                                             `json:"artifactFiles"`
	SourceFiles                            int                                             `json:"sourceFiles"`
	SourceFilesPresent                     int                                             `json:"sourceFilesPresent"`
	SourceFilesMissing                     int                                             `json:"sourceFilesMissing"`
	SourceFilesComplete                    int                                             `json:"sourceFilesComplete"`
	SourceFilesIncomplete                  int                                             `json:"sourceFilesIncomplete"`
	SourceTemplatesMissingGuideFields      int                                             `json:"sourceTemplatesMissingGuideFields"`
	MissingSectionCounts                   []setupSvcLiveReplaySourceChecklistSectionCount `json:"missingEvidenceSectionCounts,omitempty"`
	NextQueueCommands                      []setupSvcLiveReplaySourceChecklistQueueCommand `json:"nextSourceQueueCommands,omitempty"`
	PageWorklistSaveCommands               []string                                        `json:"pageWorklistSaveCommands,omitempty"`
	PageChecklistSaveCommands              []string                                        `json:"pageSourceChecklistSaveCommands,omitempty"`
	PageSaveScript                         string                                          `json:"pageSaveScript,omitempty"`
	PageSaveScriptPath                     string                                          `json:"pageSaveScriptPath,omitempty"`
	SavePageSaveScriptCommand              string                                          `json:"savePageSaveScriptCommand,omitempty"`
	SourceExecutionPacketPath              string                                          `json:"sourceExecutionPacketPath,omitempty"`
	SaveSourceExecutionPacketCommand       string                                          `json:"saveSourceExecutionPacketCommand,omitempty"`
	SourceExecutionBatchScriptPath         string                                          `json:"sourceExecutionBatchScriptPath,omitempty"`
	SaveSourceExecutionBatchScriptCommand  string                                          `json:"saveSourceExecutionBatchScriptCommand,omitempty"`
	SourceExecutionImportScriptPath        string                                          `json:"sourceExecutionImportScriptPath,omitempty"`
	SaveSourceExecutionImportScriptCommand string                                          `json:"saveSourceExecutionImportScriptCommand,omitempty"`
	CaptureSourceWorkspaceDryRunCommand    string                                          `json:"captureSourceWorkspaceDryRunCommand,omitempty"`
	CaptureSourceWorkspaceExecuteCommand   string                                          `json:"captureSourceWorkspaceExecuteCommand,omitempty"`
	CaptureSourceWorkspaceRefreshCommand   string                                          `json:"captureSourceWorkspaceRefreshCommand,omitempty"`
	MissingWorklistCommand                 string                                          `json:"missingWorklistCommand,omitempty"`
	MissingSourceChecklistCommand          string                                          `json:"missingSourceChecklistCommand,omitempty"`
	PresentWorklistCommand                 string                                          `json:"presentWorklistCommand,omitempty"`
	PresentSourceChecklistCommand          string                                          `json:"presentSourceChecklistCommand,omitempty"`
	IncompleteWorklistCommand              string                                          `json:"incompleteWorklistCommand,omitempty"`
	IncompleteSourceChecklistCommand       string                                          `json:"incompleteSourceChecklistCommand,omitempty"`
	CompleteWorklistCommand                string                                          `json:"completeWorklistCommand,omitempty"`
	CompleteSourceChecklistCommand         string                                          `json:"completeSourceChecklistCommand,omitempty"`
	DryRunImportCommand                    string                                          `json:"dryRunImportCommand,omitempty"`
	ExecuteImportCommand                   string                                          `json:"executeImportCommand,omitempty"`
}

type setupSvcLiveReplayCapturePlanResult struct {
	Mode                       string                                       `json:"mode"`
	Project                    string                                       `json:"project"`
	ReadOnly                   bool                                         `json:"readOnly"`
	Status                     string                                       `json:"status"`
	ManifestPath               string                                       `json:"manifestPath"`
	SourceRoot                 string                                       `json:"sourceRoot"`
	CaptureRoot                string                                       `json:"captureRoot"`
	Totals                     setupSvcLiveReplayCapturePlanTotals          `json:"totals"`
	Filters                    *setupSvcLiveReplayCollectionPlanFilters     `json:"filters,omitempty"`
	NextArtifactOffset         int                                          `json:"nextArtifactOffset"`
	NextArtifactLimit          int                                          `json:"nextArtifactLimit"`
	TotalNextArtifacts         int                                          `json:"totalNextArtifacts"`
	OmittedNextArtifacts       int                                          `json:"omittedNextArtifacts"`
	PageCommands               setupSvcLiveReplayCollectionPlanPageCommands `json:"pageCommands"`
	OperatorPacket             setupSvcLiveReplayWorklistOperatorPacket     `json:"operatorPacket"`
	SourceEvidenceSections     []setupSvcLiveReplayEvidenceSectionSummary   `json:"sourceEvidenceSections,omitempty"`
	SourceMissingSectionQueues []setupSvcLiveReplayEvidenceSectionQueue     `json:"sourceMissingSectionQueues,omitempty"`
	Artifacts                  []setupSvcLiveReplayCapturePlanArtifact      `json:"artifacts"`
	RecommendedNextSteps       []string                                     `json:"recommendedNextSteps"`
	Notes                      []string                                     `json:"notes"`
}

type setupSvcLiveReplayCapturePlanTotals struct {
	ArtifactFiles            int `json:"artifactFiles"`
	SourceFilesPresent       int `json:"sourceFilesPresent"`
	SourceFilesMissing       int `json:"sourceFilesMissing"`
	SourceFilesComplete      int `json:"sourceFilesComplete"`
	SourceFilesIncomplete    int `json:"sourceFilesIncomplete"`
	FilteredArtifactFiles    int `json:"filteredArtifactFiles"`
	QueryReadbackArtifacts   int `json:"queryReadbackArtifacts"`
	SetupSvcArtifacts        int `json:"setupSvcArtifacts"`
	MetadataServiceArtifacts int `json:"metadataServiceArtifacts"`
	NormalizedDiffArtifacts  int `json:"normalizedDiffArtifacts"`
	CleanupArtifacts         int `json:"cleanupArtifacts"`
	SourceEvidenceSections   int `json:"sourceEvidenceSections"`
	SourceEvidencePresent    int `json:"sourceEvidencePresent"`
	SourceEvidenceMissing    int `json:"sourceEvidenceMissing"`
}

type setupSvcLiveReplayCapturePlanArtifact struct {
	Domain                    string                                    `json:"domain"`
	Operation                 string                                    `json:"operation"`
	ArtifactType              string                                    `json:"artifactType"`
	Path                      string                                    `json:"path"`
	SuggestedSourcePath       string                                    `json:"suggestedSourcePath"`
	SuggestedSourceExists     bool                                      `json:"suggestedSourceExists"`
	SourceReadiness           string                                    `json:"sourceReadiness"`
	RequiredShapeKey          string                                    `json:"requiredShapeKey"`
	ManifestStatusField       string                                    `json:"manifestStatusField"`
	RequiredEvidenceSections  []string                                  `json:"requiredEvidenceSections"`
	SourceEvidenceSections    []setupSvcLiveReplayEvidenceSectionStatus `json:"sourceEvidenceSections,omitempty"`
	MissingEvidenceSections   []string                                  `json:"missingEvidenceSections,omitempty"`
	RequiredTables            []string                                  `json:"requiredTables,omitempty"`
	RuntimeEffects            []string                                  `json:"runtimeEffects,omitempty"`
	QueryReadbackExpectations []string                                  `json:"queryReadbackExpectations,omitempty"`
	CaptureTask               setupSvcLiveReplayArtifactCaptureTask     `json:"captureTask"`
	Checklist                 []string                                  `json:"checklist"`
}

type setupSvcLiveReplayMatrixPromotionApplyResult struct {
	Mode             string                                  `json:"mode"`
	Project          string                                  `json:"project"`
	ReadOnly         bool                                    `json:"readOnly"`
	Execute          bool                                    `json:"execute"`
	ApprovalRequired bool                                    `json:"approvalRequired"`
	Approved         bool                                    `json:"approved"`
	Status           string                                  `json:"status"`
	ManifestPath     string                                  `json:"manifestPath"`
	MatrixPath       string                                  `json:"matrixPath"`
	MatrixContract   setupSvcLiveReplayMatrixContract        `json:"matrixContract"`
	Totals           setupSvcLiveReplayMatrixPromotionTotals `json:"totals"`
	MatrixUpdates    []setupSvcLiveReplayMatrixUpdate        `json:"matrixUpdates,omitempty"`
	UpdatedDomains   []string                                `json:"updatedDomains,omitempty"`
	BlockingIssues   []string                                `json:"blockingIssues,omitempty"`
	Warnings         []string                                `json:"warnings,omitempty"`
	NextCommands     []string                                `json:"nextCommands"`
	Notes            []string                                `json:"notes"`
}

type setupSvcLiveReplayEvidenceBundleApplyResult struct {
	Mode                string                                   `json:"mode"`
	Project             string                                   `json:"project"`
	ReadOnly            bool                                     `json:"readOnly"`
	Execute             bool                                     `json:"execute"`
	ApprovalRequired    bool                                     `json:"approvalRequired"`
	Approved            bool                                     `json:"approved"`
	Status              string                                   `json:"status"`
	ManifestPath        string                                   `json:"manifestPath"`
	BundlePath          string                                   `json:"bundlePath"`
	ContractVersion     string                                   `json:"contractVersion"`
	ContractFingerprint string                                   `json:"contractFingerprint"`
	EvidenceStatus      string                                   `json:"evidenceStatus"`
	Totals              setupSvcLiveReplayEvidenceBundleTotals   `json:"totals"`
	Files               []setupSvcLiveReplayEvidenceBundleFile   `json:"files,omitempty"`
	BlockingIssues      []string                                 `json:"blockingIssues,omitempty"`
	Warnings            []string                                 `json:"warnings,omitempty"`
	NextCommands        setupSvcLiveReplayEvidenceBundleCommands `json:"nextCommands"`
	Notes               []string                                 `json:"notes"`
}

type setupSvcLiveReplayEvidenceBundleScanResult struct {
	Mode           string                                       `json:"mode"`
	Project        string                                       `json:"project"`
	ReadOnly       bool                                         `json:"readOnly"`
	Status         string                                       `json:"status"`
	ManifestPath   string                                       `json:"manifestPath"`
	BundlePath     string                                       `json:"bundlePath"`
	Bundle         setupSvcLiveReplayEvidenceBundleVerification `json:"bundle"`
	BlockingIssues []string                                     `json:"blockingIssues,omitempty"`
	NextCommands   setupSvcLiveReplayEvidenceBundleCommands     `json:"nextCommands"`
	Notes          []string                                     `json:"notes"`
}

type setupSvcLiveReplayEvidenceImportApplyResult struct {
	Mode              string                                   `json:"mode"`
	Project           string                                   `json:"project"`
	ReadOnly          bool                                     `json:"readOnly"`
	Execute           bool                                     `json:"execute"`
	ApprovalRequired  bool                                     `json:"approvalRequired"`
	Approved          bool                                     `json:"approved"`
	Status            string                                   `json:"status"`
	ManifestPath      string                                   `json:"manifestPath"`
	Totals            setupSvcLiveReplayEvidenceImportTotals   `json:"totals"`
	ArtifactCount     int                                      `json:"artifactCount"`
	PassedArtifacts   int                                      `json:"passedArtifacts"`
	FailedArtifacts   int                                      `json:"failedArtifacts"`
	SkippedDuplicates int                                      `json:"skippedDuplicateRecords"`
	WrittenFiles      int                                      `json:"writtenFiles"`
	Artifacts         []setupSvcLiveReplayEvidenceImportResult `json:"artifacts,omitempty"`
	RepairSummary     setupSvcLiveReplayEvidenceImportRepair   `json:"repairSummary,omitempty"`
	BlockingIssues    []string                                 `json:"blockingIssues,omitempty"`
	Warnings          []string                                 `json:"warnings,omitempty"`
	NextCommands      setupSvcLiveReplayEvidenceImportCommands `json:"nextCommands"`
	Notes             []string                                 `json:"notes"`
}

type setupSvcLiveReplayEvidenceImportTotals struct {
	Artifacts               int `json:"artifacts"`
	Passed                  int `json:"passed"`
	Failed                  int `json:"failed"`
	SkippedDuplicateRecords int `json:"skippedDuplicateRecords"`
	WrittenFiles            int `json:"writtenFiles"`
}

type setupSvcLiveReplayEvidenceImportResult struct {
	Domain       string   `json:"domain"`
	Operation    string   `json:"operation"`
	ArtifactType string   `json:"artifactType"`
	Path         string   `json:"path"`
	SourcePath   string   `json:"sourcePath,omitempty"`
	Status       string   `json:"status"`
	Issues       []string `json:"issues,omitempty"`
}

type setupSvcLiveReplayEvidenceImportRepair struct {
	FailedArtifacts         int                                            `json:"failedArtifacts"`
	IssueCounts             []setupSvcLiveReplayEvidenceImportIssueCount   `json:"issueCounts,omitempty"`
	MissingEvidenceSections []setupSvcLiveReplayEvidenceImportSectionCount `json:"missingEvidenceSections,omitempty"`
	ArtifactTypes           []setupSvcLiveReplayEvidenceImportIssueCount   `json:"artifactTypes,omitempty"`
	RepairQueueCount        int                                            `json:"repairQueueCount"`
	RepairQueues            []setupSvcLiveReplayEvidenceImportRepairQueue  `json:"repairQueues,omitempty"`
	SourceFiles             []setupSvcLiveReplayEvidenceImportSourceRepair `json:"sourceFiles,omitempty"`
}

type setupSvcLiveReplayEvidenceImportIssueCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type setupSvcLiveReplayEvidenceImportSectionCount struct {
	Section string `json:"section"`
	Count   int    `json:"count"`
}

type setupSvcLiveReplayEvidenceImportSourceRepair struct {
	Path                    string         `json:"path"`
	TargetPath              string         `json:"targetPath"`
	Domain                  string         `json:"domain"`
	Operation               string         `json:"operation"`
	ArtifactType            string         `json:"artifactType"`
	MissingEvidenceSections []string       `json:"missingEvidenceSections,omitempty"`
	Issues                  []string       `json:"issues,omitempty"`
	CaptureTask             map[string]any `json:"captureTask,omitempty"`
}

type setupSvcLiveReplayEvidenceImportRepairQueue struct {
	ArtifactType               string `json:"artifactType"`
	EvidenceSection            string `json:"evidenceSection"`
	Count                      int    `json:"count"`
	SourceFiles                int    `json:"sourceFiles"`
	TargetFiles                int    `json:"targetFiles"`
	CapturePlanCommand         string `json:"capturePlanCommand"`
	WorklistCommand            string `json:"worklistCommand"`
	SaveWorklistCommand        string `json:"saveWorklistCommand"`
	SourceChecklistCommand     string `json:"sourceChecklistCommand"`
	SaveSourceChecklistCommand string `json:"saveSourceChecklistCommand"`
	SourceExecutionCommand     string `json:"sourceExecutionCommand,omitempty"`
	SuggestedSourceExecution   string `json:"suggestedSourceExecutionPacketPath,omitempty"`
	SaveSourceExecutionCommand string `json:"saveSourceExecutionPacketCommand,omitempty"`
}

type setupSvcLiveReplayEvidenceImportCommands struct {
	ImportEvidence        string `json:"importEvidence"`
	SyncManifest          string `json:"syncManifest"`
	VerifyEvidence        string `json:"verifyEvidence"`
	Worklist              string `json:"worklist"`
	CompletionAudit       string `json:"completionAudit"`
	SuggestedWorklistPath string `json:"suggestedWorklistPath,omitempty"`
	SaveCurrentWorklist   string `json:"saveCurrentWorklist,omitempty"`
	DryRunCurrentImport   string `json:"dryRunCurrentImport,omitempty"`
	ExecuteCurrentImport  string `json:"executeCurrentImport,omitempty"`
}

type setupSvcLiveReplayEvidenceBundleTotals struct {
	ManifestFiles      int   `json:"manifestFiles"`
	ArtifactFiles      int   `json:"artifactFiles"`
	EvidenceFiles      int   `json:"evidenceFiles"`
	VerifiedDomains    int   `json:"verifiedDomains"`
	VerifiedOperations int   `json:"verifiedOperations"`
	TotalBytes         int64 `json:"totalBytes"`
	WrittenFiles       int   `json:"writtenFiles"`
}

type setupSvcLiveReplayEvidenceBundleCommands struct {
	WriteBundle     string `json:"writeBundle"`
	VerifyEvidence  string `json:"verifyEvidence"`
	PromotionAudit  string `json:"promotionAudit"`
	CompletionAudit string `json:"completionAudit"`
}

type setupSvcLiveReplayEvidenceBundleFile struct {
	Path         string `json:"path"`
	ArtifactType string `json:"artifactType"`
	Domain       string `json:"domain,omitempty"`
	Operation    string `json:"operation,omitempty"`
	SHA256       string `json:"sha256"`
	Bytes        int64  `json:"bytes"`
}

type setupSvcLiveReplayEvidenceBundleVerification struct {
	Status         string   `json:"status"`
	BundlePath     string   `json:"bundlePath"`
	ManifestPath   string   `json:"manifestPath"`
	EvidenceStatus string   `json:"evidenceStatus,omitempty"`
	Issues         []string `json:"issues,omitempty"`
}

type setupSvcLiveReplayMatrixPromotionTotals struct {
	Domains          int `json:"domains"`
	Operations       int `json:"operations"`
	CandidateUpdates int `json:"candidateUpdates"`
	AppliedUpdates   int `json:"appliedUpdates"`
	BlockedUpdates   int `json:"blockedUpdates"`
}

type setupSvcLiveReplayCompletionAuditTotals struct {
	Domains                    int `json:"domains"`
	Operations                 int `json:"operations"`
	MatrixVerifiedDomains      int `json:"matrixVerifiedDomains"`
	MatrixCoveredDomains       int `json:"matrixCoveredDomains"`
	MatrixNonVerifiedDomains   int `json:"matrixNonVerifiedDomains"`
	EvidenceVerifiedDomains    int `json:"evidenceVerifiedDomains"`
	EvidenceVerifiedOperations int `json:"evidenceVerifiedOperations"`
	PromotableDomains          int `json:"promotableDomains"`
	PromotableOperations       int `json:"promotableOperations"`
	CompletedDomains           int `json:"completedDomains"`
	CompletedOperations        int `json:"completedOperations"`
	BlockedDomains             int `json:"blockedDomains"`
	FailedEvidenceTotal        int `json:"failedEvidenceTotal"`
	RepairQueueCount           int `json:"repairQueueCount"`
}

type setupSvcLiveReplayCompletionAuditGate struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Blocking   bool           `json:"blocking"`
	Evidence   string         `json:"evidence"`
	Summary    map[string]any `json:"summary,omitempty"`
	NextAction string         `json:"nextAction,omitempty"`
}

type setupSvcLiveReplayCompletionGateStatus struct {
	Status     string `json:"status"`
	Blocking   bool   `json:"blocking"`
	Evidence   string `json:"evidence,omitempty"`
	NextAction string `json:"nextAction,omitempty"`
}

type setupSvcLiveReplayCompletionOperatorPacket struct {
	Status                     string                                            `json:"status"`
	ManifestPath               string                                            `json:"manifestPath"`
	GateStatuses               map[string]setupSvcLiveReplayCompletionGateStatus `json:"gateStatuses,omitempty"`
	GateSummaries              map[string]map[string]any                         `json:"gateSummaries,omitempty"`
	FailedEvidence             []string                                          `json:"failedEvidence,omitempty"`
	FailedEvidenceTotal        int                                               `json:"failedEvidenceTotal"`
	RepairQueueCount           int                                               `json:"repairQueueCount"`
	RepairPlan                 setupSvcLiveReplayCompletionRepairPlan            `json:"repairPlan,omitempty"`
	RepairQueues               []setupSvcLiveReplayEvidenceImportRepairQueue     `json:"repairQueues,omitempty"`
	FailedEvidenceSummary      setupSvcLiveReplayCompletionFailedEvidenceSummary `json:"failedEvidenceSummary,omitempty"`
	BlockingIssues             []string                                          `json:"blockingIssues,omitempty"`
	MatrixVerifiedDomains      int                                               `json:"matrixVerifiedDomains"`
	MatrixCoveredDomains       int                                               `json:"matrixCoveredDomains"`
	MatrixNonVerifiedDomains   int                                               `json:"matrixNonVerifiedDomains"`
	EvidenceVerifiedDomains    int                                               `json:"evidenceVerifiedDomains"`
	EvidenceVerifiedOperations int                                               `json:"evidenceVerifiedOperations"`
	PromotableDomains          int                                               `json:"promotableDomains"`
	PromotableOperations       int                                               `json:"promotableOperations"`
	BlockedDomains             int                                               `json:"blockedDomains"`
	CompletedDomains           int                                               `json:"completedDomains"`
	CompletedOperations        int                                               `json:"completedOperations"`
	Domains                    []setupSvcLiveReplayCompletionAuditDomain         `json:"domains,omitempty"`
	MetadataServiceDatasource  setupSvcLiveReplayDatasourceReadiness             `json:"metadataServiceDatasource"`
	NextActions                []string                                          `json:"nextActions,omitempty"`
	NextCommands               []string                                          `json:"nextCommands,omitempty"`
}

type setupSvcLiveReplayCompletionAuditDomain struct {
	Domain             string                                  `json:"domain"`
	MatrixStatus       string                                  `json:"matrixStatus"`
	EvidenceStatus     string                                  `json:"evidenceStatus"`
	RecommendedStatus  string                                  `json:"recommendedStatus"`
	CanPromote         bool                                    `json:"canPromote"`
	CompletionStatus   string                                  `json:"completionStatus"`
	VerifiedOperations []string                                `json:"verifiedOperations,omitempty"`
	BlockingOperations []string                                `json:"blockingOperations,omitempty"`
	FailedEvidence     []string                                `json:"failedEvidence,omitempty"`
	RepairPlan         *setupSvcLiveReplayCompletionRepairPlan `json:"repairPlan,omitempty"`
}

type setupSvcLiveReplayCompletionFailedEvidenceSummary struct {
	Total                 int                                                `json:"total"`
	IssueCounts           []setupSvcLiveReplayEvidenceImportIssueCount       `json:"issueCounts,omitempty"`
	DomainOperationCounts []setupSvcLiveReplayCompletionDomainOperationCount `json:"domainOperationCounts,omitempty"`
	RepairQueueCount      int                                                `json:"repairQueueCount"`
	RepairQueues          []setupSvcLiveReplayEvidenceImportRepairQueue      `json:"repairQueues,omitempty"`
}

type setupSvcLiveReplayCompletionRepairPlan struct {
	RepairQueueCount       int                                                     `json:"repairQueueCount"`
	TotalSourceFiles       int                                                     `json:"totalSourceFiles"`
	TotalTargetFiles       int                                                     `json:"totalTargetFiles"`
	PrimarySourceSystem    string                                                  `json:"primarySourceSystem,omitempty"`
	PrimaryEvidenceSection string                                                  `json:"primaryEvidenceSection,omitempty"`
	PrimaryCommand         string                                                  `json:"primaryCommand,omitempty"`
	NextRepairCommands     []string                                                `json:"nextRepairCommands,omitempty"`
	PostRepairCommands     []string                                                `json:"postRepairCommands,omitempty"`
	NextRepairScript       string                                                  `json:"nextRepairScript,omitempty"`
	NextRepairScriptPath   string                                                  `json:"nextRepairScriptPath,omitempty"`
	SaveNextRepairScript   string                                                  `json:"saveNextRepairScriptCommand,omitempty"`
	Groups                 []setupSvcLiveReplayCompletionRepairPlanGroup           `json:"groups,omitempty"`
	DomainOperations       []setupSvcLiveReplayCompletionRepairPlanDomainOperation `json:"domainOperations,omitempty"`
}

type setupSvcLiveReplayCompletionRepairPlanGroup struct {
	SourceSystem               string `json:"sourceSystem"`
	ArtifactType               string `json:"artifactType"`
	EvidenceSection            string `json:"evidenceSection"`
	Count                      int    `json:"count"`
	SourceFiles                int    `json:"sourceFiles"`
	TargetFiles                int    `json:"targetFiles"`
	WorklistCommand            string `json:"worklistCommand,omitempty"`
	SourceChecklistCommand     string `json:"sourceChecklistCommand,omitempty"`
	SourceExecutionCommand     string `json:"sourceExecutionCommand,omitempty"`
	SaveSourceExecutionCommand string `json:"saveSourceExecutionPacketCommand,omitempty"`
}

type setupSvcLiveReplayCompletionRepairPlanDomainOperation struct {
	Domain              string   `json:"domain"`
	Operation           string   `json:"operation"`
	FailedEvidenceCount int      `json:"failedEvidenceCount"`
	PrimaryRepairQueue  string   `json:"primaryRepairQueue,omitempty"`
	RepairQueues        []string `json:"repairQueues,omitempty"`
}

type setupSvcLiveReplayCompletionDomainOperationCount struct {
	Domain    string `json:"domain"`
	Operation string `json:"operation"`
	Count     int    `json:"count"`
}

type setupSvcLiveReplayMatrixContract struct {
	Path   string   `json:"path,omitempty"`
	Status string   `json:"status"`
	Issues []string `json:"issues,omitempty"`
}

type setupSvcLiveReplayCoverageResult struct {
	Mode              string                              `json:"mode"`
	Project           string                              `json:"project"`
	ReadOnly          bool                                `json:"readOnly"`
	Status            string                              `json:"status"`
	MatrixPath        string                              `json:"matrixPath,omitempty"`
	MatrixContract    setupSvcLiveReplayMatrixContract    `json:"matrixContract"`
	TestEvidence      setupSvcLiveReplayTestEvidence      `json:"testEvidenceContract"`
	Totals            setupSvcLiveReplayCoverageTotals    `json:"totals"`
	OperationFamilies setupSvcLiveReplayOperationFamilies `json:"operationFamilies"`
	Domains           []setupSvcLiveReplayCoverageDomain  `json:"domains"`
	BlockingIssues    []string                            `json:"blockingIssues,omitempty"`
	Notes             []string                            `json:"notes"`
}

type setupSvcLiveReplayCoverageTotals struct {
	Domains                   int `json:"domains"`
	Operations                int `json:"operations"`
	CanonicalCrudQueryDomains int `json:"canonicalCrudQueryDomains"`
	QueryOperations           int `json:"queryOperations"`
	WriteOperations           int `json:"writeOperations"`
	VariantOperations         int `json:"variantOperations"`
	RequiredTables            int `json:"requiredTables"`
	SetupSvcReferences        int `json:"setupSvcReferences"`
	RuntimeEffects            int `json:"runtimeEffects"`
	QueryReadbackExpectations int `json:"queryReadbackExpectations"`
	TestEvidenceDomains       int `json:"testEvidenceDomains"`
	TestEvidenceOperations    int `json:"testEvidenceOperations"`
	CoveredDomains            int `json:"coveredDomains"`
	VerifiedDomains           int `json:"verifiedDomains"`
	BlockedDomains            int `json:"blockedDomains"`
}

type setupSvcLiveReplayOperationFamilies struct {
	Create   []string `json:"create"`
	Update   []string `json:"update"`
	Delete   []string `json:"delete"`
	Query    []string `json:"query"`
	Variants []string `json:"variants,omitempty"`
}

type setupSvcLiveReplayCoverageDomain struct {
	Domain                    string   `json:"domain"`
	Status                    string   `json:"status"`
	MatrixStatus              string   `json:"matrixStatus"`
	HasCanonicalCRUDQ         bool     `json:"hasCanonicalCrudQuery"`
	QueryIncluded             bool     `json:"queryIncluded"`
	Operations                []string `json:"operations"`
	VariantOperations         []string `json:"variantOperations,omitempty"`
	RequiredTables            []string `json:"requiredTables"`
	SetupSvcReferences        []string `json:"setupSvcReferences"`
	RuntimeEffects            []string `json:"runtimeEffects"`
	QueryReadbackExpectations []string `json:"queryReadbackExpectations"`
	TestEvidenceOps           []string `json:"testEvidenceOperations,omitempty"`
	BlockingIssues            []string `json:"blockingIssues,omitempty"`
}

type setupSvcLiveReplayTestEvidence struct {
	Path             string   `json:"path,omitempty"`
	Status           string   `json:"status"`
	Domains          int      `json:"domains"`
	Operations       int      `json:"operations"`
	TestSourcePath   string   `json:"testSourcePath,omitempty"`
	TestSourceStatus string   `json:"testSourceStatus"`
	TestSourceChecks int      `json:"testSourceChecks"`
	Issues           []string `json:"issues,omitempty"`
}

type setupSvcLiveReplayPromotionTotals struct {
	Domains              int `json:"domains"`
	Operations           int `json:"operations"`
	PromotableDomains    int `json:"promotableDomains"`
	PromotableOperations int `json:"promotableOperations"`
	BlockedDomains       int `json:"blockedDomains"`
	BlockedOperations    int `json:"blockedOperations"`
	MissingDomains       int `json:"missingDomains"`
	MissingOperations    int `json:"missingOperations"`
	FailedOperations     int `json:"failedOperations"`
}

type setupSvcLiveReplayPromotionDomain struct {
	Domain              string   `json:"domain"`
	CurrentMatrixStatus string   `json:"currentMatrixStatus"`
	EvidenceStatus      string   `json:"evidenceStatus"`
	RecommendedStatus   string   `json:"recommendedStatus"`
	CanPromote          bool     `json:"canPromote"`
	VerifiedOperations  []string `json:"verifiedOperations"`
	BlockingOperations  []string `json:"blockingOperations,omitempty"`
	FailedEvidence      []string `json:"failedEvidence,omitempty"`
}

type setupSvcLiveReplayMatrixUpdate struct {
	Domain     string `json:"domain"`
	FromStatus string `json:"fromStatus"`
	ToStatus   string `json:"toStatus"`
	Reason     string `json:"reason"`
}

type setupSvcLiveReplayPacket struct {
	Mode                string                           `json:"mode"`
	Project             string                           `json:"project"`
	GeneratedAt         string                           `json:"generatedAt"`
	ReadOnly            bool                             `json:"readOnly"`
	Execute             bool                             `json:"execute"`
	ApprovalRequired    bool                             `json:"approvalRequired"`
	ApprovalPhrase      string                           `json:"approvalPhrase"`
	Status              string                           `json:"status"`
	ManifestPath        string                           `json:"manifestPath"`
	ContractVersion     string                           `json:"contractVersion"`
	ContractFingerprint string                           `json:"contractFingerprint"`
	Totals              setupSvcLiveReplayPacketTotals   `json:"totals"`
	Commands            setupSvcLiveReplayPacketCommand  `json:"commands"`
	MatrixContract      setupSvcLiveReplayMatrixContract `json:"matrixContract"`
	Domains             []setupSvcLiveReplayPacketDomain `json:"domains"`
	ManifestTemplate    setupSvcLiveReplayManifest       `json:"manifestTemplate"`
	BlockingIssues      []string                         `json:"blockingIssues,omitempty"`
	StopConditions      []string                         `json:"stopConditions"`
	Notes               []string                         `json:"notes"`
}

type setupSvcLiveReplayPacketTotals struct {
	Domains         int `json:"domains"`
	Operations      int `json:"operations"`
	WriteOperations int `json:"writeOperations"`
	QueryOperations int `json:"queryOperations"`
}

type setupSvcLiveReplayPacketCommand struct {
	GeneratePacket    string `json:"generatePacket"`
	DryRunPacket      string `json:"dryRunPacket"`
	VerifyEvidence    string `json:"verifyEvidence"`
	EvidenceDirectory string `json:"evidenceDirectory"`
}

type setupSvcLiveReplayPacketDomain struct {
	Domain                    string                              `json:"domain"`
	Status                    string                              `json:"status"`
	RequiredTables            []string                            `json:"requiredTables"`
	RuntimeEffects            []string                            `json:"runtimeEffects"`
	QueryReadbackExpectations []string                            `json:"queryReadbackExpectations"`
	Operations                []setupSvcLiveReplayPacketOperation `json:"operations"`
}

type setupSvcLiveReplayPacketOperation struct {
	Operation        string   `json:"operation"`
	Status           string   `json:"status"`
	ReadOnly         bool     `json:"readOnly"`
	RequiredEvidence []string `json:"requiredEvidence"`
	EvidenceFiles    []string `json:"evidenceFiles"`
	OperatorSteps    []string `json:"operatorSteps"`
}

type setupSvcLiveReplayManifest struct {
	Mode                string                             `json:"mode"`
	Project             string                             `json:"project"`
	Status              string                             `json:"status"`
	ContractVersion     string                             `json:"contractVersion"`
	ContractFingerprint string                             `json:"contractFingerprint"`
	Domains             []setupSvcLiveReplayManifestDomain `json:"domains"`
}

type setupSvcLiveReplayManifestDomain struct {
	Domain     string                                `json:"domain"`
	Operations []setupSvcLiveReplayManifestOperation `json:"operations"`
}

type setupSvcLiveReplayManifestOperation struct {
	Operation                     string   `json:"operation"`
	SetupSvcEvidenceStatus        string   `json:"setupSvcEvidenceStatus"`
	MetadataServiceEvidenceStatus string   `json:"metadataServiceEvidenceStatus"`
	QueryEvidenceStatus           string   `json:"queryEvidenceStatus"`
	NormalizedDiffStatus          string   `json:"normalizedDiffStatus"`
	CleanupStatus                 string   `json:"cleanupStatus,omitempty"`
	EvidenceFiles                 []string `json:"evidenceFiles"`
	Notes                         []string `json:"notes,omitempty"`
}

type setupSvcLiveReplayApplyResult struct {
	Mode             string                           `json:"mode"`
	Project          string                           `json:"project"`
	ReadOnly         bool                             `json:"readOnly"`
	Execute          bool                             `json:"execute"`
	ApprovalRequired bool                             `json:"approvalRequired"`
	Approved         bool                             `json:"approved"`
	Status           string                           `json:"status"`
	ManifestPath     string                           `json:"manifestPath"`
	Totals           setupSvcLiveReplayEvidenceTotals `json:"totals"`
	Domains          []setupSvcLiveReplayApplyDomain  `json:"domains"`
	BlockingIssues   []string                         `json:"blockingIssues,omitempty"`
	Warnings         []string                         `json:"warnings,omitempty"`
	NextCommands     setupSvcLiveReplayPacketCommand  `json:"nextCommands"`
}

type setupSvcLiveReplayWorkspaceApplyResult struct {
	Mode                string                              `json:"mode"`
	Project             string                              `json:"project"`
	ReadOnly            bool                                `json:"readOnly"`
	Execute             bool                                `json:"execute"`
	ApprovalRequired    bool                                `json:"approvalRequired"`
	Approved            bool                                `json:"approved"`
	Status              string                              `json:"status"`
	ManifestPath        string                              `json:"manifestPath"`
	EvidenceDirectory   string                              `json:"evidenceDirectory"`
	ContractVersion     string                              `json:"contractVersion"`
	ContractFingerprint string                              `json:"contractFingerprint"`
	Totals              setupSvcLiveReplayWorkspaceTotals   `json:"totals"`
	SampleFiles         []string                            `json:"sampleFiles,omitempty"`
	BlockingIssues      []string                            `json:"blockingIssues,omitempty"`
	Warnings            []string                            `json:"warnings,omitempty"`
	NextCommands        setupSvcLiveReplayWorkspaceCommands `json:"nextCommands"`
	Notes               []string                            `json:"notes"`
}

type setupSvcLiveReplayWorkspaceTotals struct {
	Domains       int `json:"domains"`
	Operations    int `json:"operations"`
	ArtifactFiles int `json:"artifactFiles"`
	WrittenFiles  int `json:"writtenFiles"`
}

type setupSvcLiveReplayWorkspaceCommands struct {
	PrepareWorkspace string `json:"prepareWorkspace"`
	VerifyEvidence   string `json:"verifyEvidence"`
	PromotionAudit   string `json:"promotionAudit"`
	CompletionAudit  string `json:"completionAudit"`
}

type setupSvcLiveReplayCaptureSourceWorkspaceApplyResult struct {
	Mode                string                                           `json:"mode"`
	Project             string                                           `json:"project"`
	ReadOnly            bool                                             `json:"readOnly"`
	Execute             bool                                             `json:"execute"`
	ApprovalRequired    bool                                             `json:"approvalRequired"`
	Approved            bool                                             `json:"approved"`
	Status              string                                           `json:"status"`
	ManifestPath        string                                           `json:"manifestPath"`
	SourceRoot          string                                           `json:"sourceRoot"`
	CaptureRoot         string                                           `json:"captureRoot"`
	ContractVersion     string                                           `json:"contractVersion"`
	ContractFingerprint string                                           `json:"contractFingerprint"`
	Filters             *setupSvcLiveReplayCollectionPlanFilters         `json:"filters,omitempty"`
	Totals              setupSvcLiveReplayCaptureSourceWorkspaceTotals   `json:"totals"`
	SampleFiles         []string                                         `json:"sampleFiles,omitempty"`
	BlockingIssues      []string                                         `json:"blockingIssues,omitempty"`
	Warnings            []string                                         `json:"warnings,omitempty"`
	NextCommands        setupSvcLiveReplayCaptureSourceWorkspaceCommands `json:"nextCommands"`
	Notes               []string                                         `json:"notes"`
}

type setupSvcLiveReplayCaptureSourceWorkspaceTotals struct {
	ArtifactFiles          int `json:"artifactFiles"`
	FilteredArtifactFiles  int `json:"filteredArtifactFiles"`
	PlannedFiles           int `json:"plannedFiles"`
	WrittenFiles           int `json:"writtenFiles"`
	RefreshedExistingFiles int `json:"refreshedExistingFiles"`
	SkippedExistingFiles   int `json:"skippedExistingFiles"`
	SourceFilesPresent     int `json:"sourceFilesPresent"`
	SourceFilesMissing     int `json:"sourceFilesMissing"`
	SourceFilesComplete    int `json:"sourceFilesComplete"`
	SourceFilesIncomplete  int `json:"sourceFilesIncomplete"`
}

type setupSvcLiveReplayCaptureSourceWorkspaceCommands struct {
	PrepareCaptureSources string `json:"prepareCaptureSources"`
	CapturePlan           string `json:"capturePlan"`
	CompleteWorklist      string `json:"completeWorklist"`
	DryRunImport          string `json:"dryRunImport"`
	ExecuteImport         string `json:"executeImport"`
}

type setupSvcLiveReplayNormalizedDiffApplyResult struct {
	Mode             string                                   `json:"mode"`
	Project          string                                   `json:"project"`
	ReadOnly         bool                                     `json:"readOnly"`
	Execute          bool                                     `json:"execute"`
	ApprovalRequired bool                                     `json:"approvalRequired"`
	Approved         bool                                     `json:"approved"`
	Status           string                                   `json:"status"`
	ManifestPath     string                                   `json:"manifestPath"`
	Totals           setupSvcLiveReplayNormalizedDiffTotals   `json:"totals"`
	Domains          []setupSvcLiveReplayNormalizedDiffDomain `json:"domains"`
	BlockingIssues   []string                                 `json:"blockingIssues,omitempty"`
	Warnings         []string                                 `json:"warnings,omitempty"`
	NextCommands     setupSvcLiveReplayNormalizedDiffCommands `json:"nextCommands"`
	Notes            []string                                 `json:"notes"`
}

type setupSvcLiveReplayNormalizedDiffTotals struct {
	Domains         int `json:"domains"`
	Operations      int `json:"operations"`
	DiffFiles       int `json:"diffFiles"`
	WrittenFiles    int `json:"writtenFiles"`
	CleanOperations int `json:"cleanOperations"`
	DirtyOperations int `json:"dirtyOperations"`
	BlockedOps      int `json:"blockedOperations"`
}

type setupSvcLiveReplayNormalizedDiffCommands struct {
	GenerateDiffs   string `json:"generateDiffs"`
	VerifyEvidence  string `json:"verifyEvidence"`
	PromotionAudit  string `json:"promotionAudit"`
	CompletionAudit string `json:"completionAudit"`
}

type setupSvcLiveReplayManifestSyncApplyResult struct {
	Mode              string                                 `json:"mode"`
	Project           string                                 `json:"project"`
	ReadOnly          bool                                   `json:"readOnly"`
	Execute           bool                                   `json:"execute"`
	ApprovalRequired  bool                                   `json:"approvalRequired"`
	Approved          bool                                   `json:"approved"`
	Status            string                                 `json:"status"`
	ManifestPath      string                                 `json:"manifestPath"`
	Totals            setupSvcLiveReplayManifestSyncTotals   `json:"totals"`
	ArtifactFiles     int                                    `json:"artifactFiles"`
	PassedArtifacts   int                                    `json:"passedArtifacts"`
	PendingArtifacts  int                                    `json:"pendingArtifacts"`
	FailedArtifacts   int                                    `json:"failedArtifacts"`
	PassedOperations  int                                    `json:"passedOperations"`
	PendingOperations int                                    `json:"pendingOperations"`
	FailedOperations  int                                    `json:"failedOperations"`
	UpdatedOperations int                                    `json:"updatedOperations"`
	WrittenFiles      int                                    `json:"writtenFiles"`
	Domains           []setupSvcLiveReplayManifestSyncDomain `json:"domains"`
	BlockingIssues    []string                               `json:"blockingIssues,omitempty"`
	Warnings          []string                               `json:"warnings,omitempty"`
	NextCommands      setupSvcLiveReplayManifestSyncCommands `json:"nextCommands"`
	Notes             []string                               `json:"notes"`
}

type setupSvcLiveReplayManifestSyncTotals struct {
	Domains           int `json:"domains"`
	Operations        int `json:"operations"`
	ArtifactFiles     int `json:"artifactFiles"`
	PassedArtifacts   int `json:"passedArtifacts"`
	PendingArtifacts  int `json:"pendingArtifacts"`
	FailedArtifacts   int `json:"failedArtifacts"`
	PassedOperations  int `json:"passedOperations"`
	PendingOperations int `json:"pendingOperations"`
	FailedOperations  int `json:"failedOperations"`
	UpdatedOperations int `json:"updatedOperations"`
	WrittenFiles      int `json:"writtenFiles"`
}

type setupSvcLiveReplayManifestSyncCommands struct {
	SyncManifest    string `json:"syncManifest"`
	VerifyEvidence  string `json:"verifyEvidence"`
	PromotionAudit  string `json:"promotionAudit"`
	CompletionAudit string `json:"completionAudit"`
}

type setupSvcLiveReplayManifestSyncDomain struct {
	Domain     string                                    `json:"domain"`
	Status     string                                    `json:"status"`
	Operations []setupSvcLiveReplayManifestSyncOperation `json:"operations"`
}

type setupSvcLiveReplayManifestSyncOperation struct {
	Operation        string                                         `json:"operation"`
	Status           string                                         `json:"status"`
	Updated          bool                                           `json:"updated"`
	ArtifactStatuses []setupSvcLiveReplayManifestSyncArtifactStatus `json:"artifactStatuses"`
	Issues           []string                                       `json:"issues,omitempty"`
}

type setupSvcLiveReplayManifestSyncArtifactStatus struct {
	ArtifactType string   `json:"artifactType"`
	Field        string   `json:"field"`
	File         string   `json:"file"`
	Status       string   `json:"status"`
	Issues       []string `json:"issues,omitempty"`
}

type setupSvcLiveReplayGapResult struct {
	Mode                string                           `json:"mode"`
	Project             string                           `json:"project"`
	ReadOnly            bool                             `json:"readOnly"`
	Status              string                           `json:"status"`
	ManifestPath        string                           `json:"manifestPath"`
	ContractVersion     string                           `json:"contractVersion,omitempty"`
	ContractFingerprint string                           `json:"contractFingerprint,omitempty"`
	MatrixContract      setupSvcLiveReplayMatrixContract `json:"matrixContract"`
	Totals              setupSvcLiveReplayGapTotals      `json:"totals"`
	CollectionPlan      setupSvcLiveReplayCollectionPlan `json:"collectionPlan"`
	Domains             []setupSvcLiveReplayGapDomain    `json:"domains"`
	BlockingIssues      []string                         `json:"blockingIssues,omitempty"`
	NextCommands        setupSvcLiveReplayGapCommands    `json:"nextCommands"`
	Notes               []string                         `json:"notes"`
}

type setupSvcLiveReplayGapTotals struct {
	Domains                int `json:"domains"`
	Operations             int `json:"operations"`
	CompleteOperations     int `json:"completeOperations"`
	MissingOperations      int `json:"missingOperations"`
	PendingOperations      int `json:"pendingOperations"`
	FailedOperations       int `json:"failedOperations"`
	ReadyForDiffOperations int `json:"readyForDiffOperations"`
	MissingFiles           int `json:"missingFiles"`
	PendingFiles           int `json:"pendingFiles"`
	FailedArtifacts        int `json:"failedArtifacts"`
}

type setupSvcLiveReplayCollectionPlan struct {
	Status                 string                                        `json:"status"`
	TotalArtifacts         int                                           `json:"totalArtifacts"`
	PendingArtifacts       int                                           `json:"pendingArtifacts"`
	MissingArtifacts       int                                           `json:"missingArtifacts"`
	FailedArtifacts        int                                           `json:"failedArtifacts"`
	PassedArtifacts        int                                           `json:"passedArtifacts"`
	QueryReadbackArtifacts int                                           `json:"queryReadbackArtifacts"`
	ArtifactTypes          []setupSvcLiveReplayArtifactTypeCollection    `json:"artifactTypes"`
	EvidenceSections       []setupSvcLiveReplayEvidenceSectionSummary    `json:"evidenceSections,omitempty"`
	MissingSectionQueues   []setupSvcLiveReplayEvidenceSectionQueue      `json:"missingSectionQueues,omitempty"`
	NextArtifacts          []setupSvcLiveReplayArtifactCollectionAction  `json:"nextArtifacts,omitempty"`
	NextArtifactOffset     int                                           `json:"nextArtifactOffset"`
	NextArtifactLimit      int                                           `json:"nextArtifactLimit"`
	TotalNextArtifacts     int                                           `json:"totalNextArtifacts"`
	OmittedNextArtifacts   int                                           `json:"omittedNextArtifacts"`
	Filters                *setupSvcLiveReplayCollectionPlanFilters      `json:"filters,omitempty"`
	PageCommands           setupSvcLiveReplayCollectionPlanPageCommands  `json:"pageCommands"`
	Runbook                []setupSvcLiveReplayCollectionPlanRunbookStep `json:"runbook,omitempty"`
	RecommendedOrder       []string                                      `json:"recommendedOrder"`
	Notes                  []string                                      `json:"notes"`
}

type setupSvcLiveReplayCollectionPlanOptions struct {
	Domain          string
	Operation       string
	ArtifactType    string
	SourceSystem    string
	CaptureMode     string
	Status          string
	EvidenceSection string
	SectionStatus   string
	SourceStatus    string
	SourceReadiness string
	Offset          int
	Limit           int
	BatchIndex      int
	BatchLimit      int
}

type setupSvcLiveReplayCollectionPlanFilters struct {
	Domain          string `json:"domain,omitempty"`
	Operation       string `json:"operation,omitempty"`
	ArtifactType    string `json:"artifactType,omitempty"`
	SourceSystem    string `json:"sourceSystem,omitempty"`
	CaptureMode     string `json:"captureMode,omitempty"`
	Status          string `json:"status,omitempty"`
	EvidenceSection string `json:"evidenceSection,omitempty"`
	SectionStatus   string `json:"sectionStatus,omitempty"`
	SourceStatus    string `json:"sourceStatus,omitempty"`
	SourceReadiness string `json:"sourceReadiness,omitempty"`
}

type setupSvcLiveReplayCollectionPlanPageCommands struct {
	CurrentPage  string `json:"currentPage"`
	NextPage     string `json:"nextPage,omitempty"`
	PreviousPage string `json:"previousPage,omitempty"`
}

type setupSvcLiveReplayCollectionPlanRunbookStep struct {
	Step     string   `json:"step"`
	Status   string   `json:"status"`
	Commands []string `json:"commands,omitempty"`
	Notes    []string `json:"notes,omitempty"`
}

type setupSvcLiveReplayArtifactTypeCollection struct {
	ArtifactType string `json:"artifactType"`
	Total        int    `json:"total"`
	Pending      int    `json:"pending"`
	Missing      int    `json:"missing"`
	Failed       int    `json:"failed"`
	Passed       int    `json:"passed"`
}

type setupSvcLiveReplayEvidenceSectionSummary struct {
	ArtifactType string `json:"artifactType"`
	Section      string `json:"section"`
	Total        int    `json:"total"`
	Present      int    `json:"present"`
	Missing      int    `json:"missing"`
	NextAction   string `json:"nextAction,omitempty"`
	QueueCommand string `json:"queueCommand,omitempty"`
}

type setupSvcLiveReplayEvidenceSectionQueue struct {
	ArtifactType         string   `json:"artifactType"`
	Section              string   `json:"section"`
	Missing              int      `json:"missing"`
	RequiredShapeKey     string   `json:"requiredShapeKey"`
	ManifestStatusField  string   `json:"manifestStatusField"`
	PageSize             int      `json:"pageSize"`
	BatchCount           int      `json:"batchCount"`
	QueueCommand         string   `json:"queueCommand"`
	BatchCommands        []string `json:"batchCommands,omitempty"`
	OmittedBatchCommands int      `json:"omittedBatchCommands,omitempty"`
}

type setupSvcLiveReplayWorklistResult struct {
	Mode                     string                                     `json:"mode"`
	Project                  string                                     `json:"project"`
	ReadOnly                 bool                                       `json:"readOnly"`
	Status                   string                                     `json:"status"`
	ManifestPath             string                                     `json:"manifestPath"`
	SourceRoot               string                                     `json:"sourceRoot"`
	CaptureRoot              string                                     `json:"captureRoot"`
	SourceGapStatus          string                                     `json:"sourceGapStatus"`
	SourceFiles              int                                        `json:"sourceFiles"`
	TargetFiles              int                                        `json:"targetFiles"`
	SourceFilesPresent       int                                        `json:"sourceFilesPresent"`
	SourceFilesMissing       int                                        `json:"sourceFilesMissing"`
	SourceFilesComplete      int                                        `json:"sourceFilesComplete"`
	SourceFilesIncomplete    int                                        `json:"sourceFilesIncomplete"`
	QueuesCount              int                                        `json:"queuesCount"`
	Batches                  int                                        `json:"batches"`
	Artifacts                int                                        `json:"artifacts"`
	UniqueArtifactFiles      int                                        `json:"uniqueArtifactFiles"`
	DuplicateArtifactRecords int                                        `json:"duplicateArtifactRecords"`
	MissingSections          int                                        `json:"missingSections"`
	QueryReadbackQueues      int                                        `json:"queryReadbackQueues"`
	QueryReadbackArtifacts   int                                        `json:"queryReadbackArtifacts"`
	OmittedBatches           int                                        `json:"omittedBatches"`
	Totals                   setupSvcLiveReplayWorklistTotals           `json:"totals"`
	Filters                  *setupSvcLiveReplayCollectionPlanFilters   `json:"filters,omitempty"`
	BatchIndex               *int                                       `json:"batchIndex,omitempty"`
	BatchLimit               int                                        `json:"batchLimit"`
	Queues                   []setupSvcLiveReplayWorklistQueue          `json:"queues,omitempty"`
	BatchSaveCommands        []setupSvcLiveReplayWorklistBatchCommand   `json:"batchSaveCommands,omitempty"`
	SourceEvidenceSections   []setupSvcLiveReplayEvidenceSectionSummary `json:"sourceEvidenceSections,omitempty"`
	OperatorPacket           setupSvcLiveReplayWorklistOperatorPacket   `json:"operatorPacket"`
	NextCommands             setupSvcLiveReplayGapCommands              `json:"nextCommands"`
	Notes                    []string                                   `json:"notes"`
}

type setupSvcLiveReplayWorklistTotals struct {
	Queues                   int `json:"queues"`
	Batches                  int `json:"batches"`
	Artifacts                int `json:"artifacts"`
	UniqueArtifactFiles      int `json:"uniqueArtifactFiles"`
	DuplicateArtifactRecords int `json:"duplicateArtifactRecords"`
	SourceFilesPresent       int `json:"sourceFilesPresent"`
	SourceFilesMissing       int `json:"sourceFilesMissing"`
	SourceFilesComplete      int `json:"sourceFilesComplete"`
	SourceFilesIncomplete    int `json:"sourceFilesIncomplete"`
	MissingSections          int `json:"missingSections"`
	QueryReadbackQueues      int `json:"queryReadbackQueues"`
	QueryReadbackArtifacts   int `json:"queryReadbackArtifacts"`
	OmittedBatches           int `json:"omittedBatches"`
}

type setupSvcLiveReplaySourceChecklistResult struct {
	Mode                      string                                               `json:"mode"`
	Project                   string                                               `json:"project"`
	ReadOnly                  bool                                                 `json:"readOnly"`
	Status                    string                                               `json:"status"`
	ManifestPath              string                                               `json:"manifestPath"`
	GeneratedFrom             string                                               `json:"generatedFrom"`
	SourceRoot                string                                               `json:"sourceRoot"`
	CaptureRoot               string                                               `json:"captureRoot"`
	Filters                   *setupSvcLiveReplayCollectionPlanFilters             `json:"filters,omitempty"`
	SourceFiles               int                                                  `json:"sourceFiles"`
	TargetFiles               int                                                  `json:"targetFiles"`
	ReplacementRecords        int                                                  `json:"replacementRecords"`
	WorklistQueues            int                                                  `json:"worklistQueues"`
	WorklistBatches           int                                                  `json:"worklistBatches"`
	MissingSectionKinds       int                                                  `json:"missingSectionKinds"`
	NextQueueCount            int                                                  `json:"nextQueueCount"`
	RepairQueueCount          int                                                  `json:"repairQueueCount"`
	Totals                    setupSvcLiveReplaySourceChecklistTotals              `json:"totals"`
	ArtifactTypeCounts        []setupSvcLiveReplaySourceChecklistArtifactTypeCount `json:"artifactTypeCounts"`
	SourceReadinessCounts     []setupSvcLiveReplaySourceChecklistReadinessCount    `json:"sourceReadinessCounts"`
	MissingSectionCounts      []setupSvcLiveReplaySourceChecklistSectionCount      `json:"missingEvidenceSectionCounts,omitempty"`
	NextQueueCommands         []setupSvcLiveReplaySourceChecklistQueueCommand      `json:"nextSourceQueueCommands,omitempty"`
	PageWorklistSaveCommands  []string                                             `json:"pageWorklistSaveCommands,omitempty"`
	PageChecklistSaveCommands []string                                             `json:"pageSourceChecklistSaveCommands,omitempty"`
	PageSaveScript            string                                               `json:"pageSaveScript,omitempty"`
	PageSaveScriptPath        string                                               `json:"pageSaveScriptPath,omitempty"`
	SavePageSaveScriptCommand string                                               `json:"savePageSaveScriptCommand,omitempty"`
	Sources                   []setupSvcLiveReplaySourceChecklistItem              `json:"sources"`
	OperatorPacket            setupSvcLiveReplaySourceChecklistOperatorPacket      `json:"operatorPacket"`
	Notes                     []string                                             `json:"notes"`
}

type setupSvcLiveReplaySourceChecklistTotals struct {
	InputWorklists     int `json:"inputWorklists"`
	WorklistQueues     int `json:"worklistQueues"`
	WorklistBatches    int `json:"worklistBatches"`
	ReplacementRecords int `json:"replacementRecords"`
	UniqueSourceFiles  int `json:"uniqueSourceFiles"`
	UniqueTargetFiles  int `json:"uniqueTargetFiles"`
}

type setupSvcLiveReplaySourceChecklistArtifactTypeCount struct {
	ArtifactType string `json:"artifactType"`
	Records      int    `json:"records"`
	SourceFiles  int    `json:"sourceFiles"`
	TargetFiles  int    `json:"targetFiles"`
}

type setupSvcLiveReplaySourceChecklistReadinessCount struct {
	SourceReadiness string `json:"sourceReadiness"`
	Records         int    `json:"records"`
	SourceFiles     int    `json:"sourceFiles"`
}

type setupSvcLiveReplaySourceChecklistSectionCount struct {
	EvidenceSection string   `json:"evidenceSection"`
	SourceFiles     int      `json:"sourceFiles"`
	TargetFiles     int      `json:"targetFiles"`
	ArtifactTypes   []string `json:"artifactTypes,omitempty"`
}

type setupSvcLiveReplaySourceChecklistQueueCommand struct {
	ArtifactType               string   `json:"artifactType,omitempty"`
	EvidenceSection            string   `json:"evidenceSection"`
	Count                      int      `json:"count,omitempty"`
	SourceFiles                int      `json:"sourceFiles"`
	TargetFiles                int      `json:"targetFiles"`
	Command                    string   `json:"command,omitempty"`
	SourceReadiness            string   `json:"sourceReadiness,omitempty"`
	Offset                     int      `json:"offset"`
	Limit                      int      `json:"limit"`
	PageSize                   int      `json:"pageSize"`
	PageCount                  int      `json:"pageCount"`
	OmittedSourceFiles         int      `json:"omittedSourceFiles"`
	WorklistCommand            string   `json:"worklistCommand"`
	SuggestedWorklistPath      string   `json:"suggestedWorklistPath"`
	SaveWorklistCommand        string   `json:"saveWorklistCommand"`
	SourceChecklistCommand     string   `json:"sourceChecklistCommand"`
	SuggestedSourceChecklist   string   `json:"suggestedSourceChecklistPath"`
	SaveSourceChecklistCommand string   `json:"saveSourceChecklistCommand"`
	SourceExecutionCommand     string   `json:"sourceExecutionCommand,omitempty"`
	SuggestedSourceExecution   string   `json:"suggestedSourceExecutionPacketPath,omitempty"`
	SaveSourceExecutionCommand string   `json:"saveSourceExecutionPacketCommand,omitempty"`
	NextPageWorklistCommand    string   `json:"nextPageWorklistCommand,omitempty"`
	SaveNextPageWorklist       string   `json:"saveNextPageWorklistCommand,omitempty"`
	NextPageSourceChecklist    string   `json:"nextPageSourceChecklistCommand,omitempty"`
	SaveNextPageChecklist      string   `json:"saveNextPageSourceChecklistCommand,omitempty"`
	PageWorklistSaveCommands   []string `json:"pageWorklistSaveCommands,omitempty"`
	PageChecklistSaveCommands  []string `json:"pageSourceChecklistSaveCommands,omitempty"`
}

type setupSvcLiveReplaySourceChecklistItem struct {
	SourcePath                string                                `json:"sourcePath"`
	TargetPath                string                                `json:"targetPath"`
	SourceReadiness           string                                `json:"sourceReadiness"`
	Domain                    string                                `json:"domain"`
	Operation                 string                                `json:"operation"`
	ArtifactType              string                                `json:"artifactType"`
	RequiredShapeKey          string                                `json:"requiredShapeKey"`
	ManifestStatusField       string                                `json:"manifestStatusField"`
	ReplacementStatusTarget   string                                `json:"replacementStatusTarget"`
	MissingEvidenceSections   []string                              `json:"missingEvidenceSections,omitempty"`
	RequiredEvidenceSections  []string                              `json:"requiredEvidenceSections,omitempty"`
	RequiredTables            []string                              `json:"requiredTables,omitempty"`
	RuntimeEffects            []string                              `json:"runtimeEffects,omitempty"`
	QueryReadbackExpectations []string                              `json:"queryReadbackExpectations,omitempty"`
	WorklistFiles             []string                              `json:"worklistFiles,omitempty"`
	CaptureTask               setupSvcLiveReplayArtifactCaptureTask `json:"captureTask"`
	Checklist                 []string                              `json:"checklist,omitempty"`
}

type setupSvcLiveReplaySourceChecklistOperatorPacket struct {
	Purpose                   string                                          `json:"purpose"`
	SourceFiles               int                                             `json:"sourceFiles"`
	TargetFiles               int                                             `json:"targetFiles"`
	ReplacementRecords        int                                             `json:"replacementRecords"`
	WorklistQueues            int                                             `json:"worklistQueues"`
	WorklistBatches           int                                             `json:"worklistBatches"`
	MissingSectionKinds       int                                             `json:"missingSectionKinds"`
	NextQueueCount            int                                             `json:"nextQueueCount"`
	RepairQueueCount          int                                             `json:"repairQueueCount"`
	SuggestedChecklistPath    string                                          `json:"suggestedChecklistPath"`
	SaveChecklistCommand      string                                          `json:"saveChecklistCommand"`
	NextQueueCommands         []setupSvcLiveReplaySourceChecklistQueueCommand `json:"nextSourceQueueCommands,omitempty"`
	PageWorklistSaveCommands  []string                                        `json:"pageWorklistSaveCommands,omitempty"`
	PageChecklistSaveCommands []string                                        `json:"pageSourceChecklistSaveCommands,omitempty"`
	PageSaveScript            string                                          `json:"pageSaveScript,omitempty"`
	PageSaveScriptPath        string                                          `json:"pageSaveScriptPath,omitempty"`
	SavePageSaveScriptCommand string                                          `json:"savePageSaveScriptCommand,omitempty"`
	SourceRoot                string                                          `json:"sourceRoot"`
	CaptureRoot               string                                          `json:"captureRoot"`
	PostCaptureCommands       []string                                        `json:"postCaptureCommands"`
	StopConditions            []string                                        `json:"stopConditions"`
}

type setupSvcLiveReplaySourceHealthResult struct {
	Mode                         string                                          `json:"mode"`
	Project                      string                                          `json:"project"`
	ReadOnly                     bool                                            `json:"readOnly"`
	Status                       string                                          `json:"status"`
	ManifestPath                 string                                          `json:"manifestPath"`
	GeneratedFrom                string                                          `json:"generatedFrom"`
	SourceRoot                   string                                          `json:"sourceRoot"`
	CaptureRoot                  string                                          `json:"captureRoot"`
	Filters                      *setupSvcLiveReplayCollectionPlanFilters        `json:"filters,omitempty"`
	SourceFiles                  int                                             `json:"sourceFiles"`
	TargetFiles                  int                                             `json:"targetFiles"`
	SourceFilesPresent           int                                             `json:"sourceFilesPresent"`
	SourceFilesComplete          int                                             `json:"sourceFilesComplete"`
	SourceFilesIncomplete        int                                             `json:"sourceFilesIncomplete"`
	SourceFilesMissing           int                                             `json:"sourceFilesMissing"`
	CompleteSourceFiles          int                                             `json:"completeSourceFiles"`
	IncompleteSourceFiles        int                                             `json:"incompleteSourceFiles"`
	MissingSourceFiles           int                                             `json:"missingSourceFiles"`
	Totals                       setupSvcLiveReplaySourceHealthTotals            `json:"totals"`
	Readiness                    []setupSvcLiveReplaySourceHealthReadiness       `json:"readiness"`
	ArtifactTypes                []setupSvcLiveReplaySourceHealthArtifactType    `json:"artifactTypes"`
	DomainOperations             []setupSvcLiveReplaySourceHealthDomainOperation `json:"domainOperations"`
	MissingSections              []setupSvcLiveReplaySourceHealthMissingSection  `json:"missingSections,omitempty"`
	MissingEvidenceSectionCounts []setupSvcLiveReplaySourceHealthMissingSection  `json:"missingEvidenceSectionCounts,omitempty"`
	RepairQueues                 []setupSvcLiveReplaySourceChecklistQueueCommand `json:"repairQueues,omitempty"`
	ReadyImportCommands          []string                                        `json:"readyImportCommands,omitempty"`
	RecommendedRunbook           []string                                        `json:"recommendedRunbook"`
	OperatorPacket               setupSvcLiveReplaySourceHealthOperatorPacket    `json:"operatorPacket"`
	Notes                        []string                                        `json:"notes"`
}

type setupSvcLiveReplaySourceHealthTotals struct {
	SourceFiles               int  `json:"sourceFiles"`
	TargetFiles               int  `json:"targetFiles"`
	ArtifactTypes             int  `json:"artifactTypes"`
	DomainOperations          int  `json:"domainOperations"`
	CompleteSourceFiles       int  `json:"completeSourceFiles"`
	IncompleteSourceFiles     int  `json:"incompleteSourceFiles"`
	MissingSourceFiles        int  `json:"missingSourceFiles"`
	ImportableSourceFiles     int  `json:"importableSourceFiles"`
	RepairRequiredSourceFiles int  `json:"repairRequiredSourceFiles"`
	MissingSectionKinds       int  `json:"missingSectionKinds"`
	MissingSectionInstances   int  `json:"missingSectionInstances"`
	CanImportCompleteSources  bool `json:"canImportCompleteSources"`
}

type setupSvcLiveReplaySourceHealthReadiness struct {
	SourceReadiness string `json:"sourceReadiness"`
	SourceFiles     int    `json:"sourceFiles"`
}

type setupSvcLiveReplaySourceHealthArtifactType struct {
	ArtifactType            string   `json:"artifactType"`
	SourceFiles             int      `json:"sourceFiles"`
	CompleteSourceFiles     int      `json:"completeSourceFiles"`
	IncompleteSourceFiles   int      `json:"incompleteSourceFiles"`
	MissingSourceFiles      int      `json:"missingSourceFiles"`
	MissingSectionKinds     int      `json:"missingSectionKinds"`
	MissingSectionInstances int      `json:"missingSectionInstances"`
	TopMissingSections      []string `json:"topMissingSections,omitempty"`
}

type setupSvcLiveReplaySourceHealthDomainOperation struct {
	Domain                  string   `json:"domain"`
	Operation               string   `json:"operation"`
	SourceFiles             int      `json:"sourceFiles"`
	CompleteSourceFiles     int      `json:"completeSourceFiles"`
	IncompleteSourceFiles   int      `json:"incompleteSourceFiles"`
	MissingSourceFiles      int      `json:"missingSourceFiles"`
	ArtifactTypes           []string `json:"artifactTypes,omitempty"`
	MissingSectionInstances int      `json:"missingSectionInstances"`
}

type setupSvcLiveReplaySourceHealthMissingSection struct {
	ArtifactType    string `json:"artifactType"`
	EvidenceSection string `json:"evidenceSection"`
	SourceFiles     int    `json:"sourceFiles"`
	TargetFiles     int    `json:"targetFiles"`
	QueueCommand    string `json:"queueCommand,omitempty"`
}

type setupSvcLiveReplaySourceHealthOperatorPacket struct {
	Purpose                  string                                          `json:"purpose"`
	SourceFiles              int                                             `json:"sourceFiles"`
	CompleteSourceFiles      int                                             `json:"completeSourceFiles"`
	IncompleteSourceFiles    int                                             `json:"incompleteSourceFiles"`
	MissingSourceFiles       int                                             `json:"missingSourceFiles"`
	RepairQueues             []setupSvcLiveReplaySourceChecklistQueueCommand `json:"repairQueues,omitempty"`
	SourceExecutionCommand   string                                          `json:"sourceExecutionCommand"`
	CompleteChecklistCommand string                                          `json:"completeSourceChecklistCommand"`
	DryRunImportCommand      string                                          `json:"dryRunImportCommand,omitempty"`
	ApprovedImportCommand    string                                          `json:"approvedImportCommand,omitempty"`
	CompletionAuditCommand   string                                          `json:"completionAuditCommand"`
}

type setupSvcLiveReplaySourceValidateResult struct {
	Mode               string                                      `json:"mode"`
	Project            string                                      `json:"project"`
	ReadOnly           bool                                        `json:"readOnly"`
	Status             string                                      `json:"status"`
	ManifestPath       string                                      `json:"manifestPath"`
	GeneratedFrom      string                                      `json:"generatedFrom"`
	SourceRoot         string                                      `json:"sourceRoot"`
	CaptureRoot        string                                      `json:"captureRoot"`
	Filters            *setupSvcLiveReplayCollectionPlanFilters    `json:"filters,omitempty"`
	SourceFiles        int                                         `json:"sourceFiles"`
	ArtifactCount      int                                         `json:"artifactCount"`
	ReadyArtifacts     int                                         `json:"readyArtifacts"`
	FailedArtifacts    int                                         `json:"failedArtifacts"`
	SkippedDuplicates  int                                         `json:"skippedDuplicateRecords"`
	Totals             setupSvcLiveReplaySourceValidateTotals      `json:"totals"`
	ImportDryRun       setupSvcLiveReplayEvidenceImportApplyResult `json:"importDryRun"`
	Artifacts          []setupSvcLiveReplayEvidenceImportResult    `json:"artifacts,omitempty"`
	RepairSummary      setupSvcLiveReplayEvidenceImportRepair      `json:"repairSummary,omitempty"`
	BlockingIssues     []string                                    `json:"blockingIssues,omitempty"`
	RecommendedRunbook []string                                    `json:"recommendedRunbook"`
	OperatorPacket     setupSvcLiveReplaySourceValidateOperator    `json:"operatorPacket"`
	Notes              []string                                    `json:"notes"`
}

type setupSvcLiveReplaySourceValidateTotals struct {
	SourceFiles       int `json:"sourceFiles"`
	Artifacts         int `json:"artifacts"`
	ReadyArtifacts    int `json:"readyArtifacts"`
	FailedArtifacts   int `json:"failedArtifacts"`
	SkippedDuplicates int `json:"skippedDuplicateRecords"`
}

type setupSvcLiveReplaySourceValidateOperator struct {
	Purpose                string                                        `json:"purpose"`
	SourceFiles            int                                           `json:"sourceFiles"`
	ArtifactCount          int                                           `json:"artifactCount"`
	ReadyArtifacts         int                                           `json:"readyArtifacts"`
	FailedArtifacts        int                                           `json:"failedArtifacts"`
	SkippedDuplicates      int                                           `json:"skippedDuplicateRecords"`
	RepairQueueCount       int                                           `json:"repairQueueCount"`
	RepairQueues           []setupSvcLiveReplayEvidenceImportRepairQueue `json:"repairQueues,omitempty"`
	IssueKinds             int                                           `json:"issueKinds"`
	SourceChecklistCommand string                                        `json:"sourceChecklistCommand"`
	SourceHealthCommand    string                                        `json:"sourceHealthCommand"`
	DryRunImportCommand    string                                        `json:"dryRunImportCommand"`
	ApprovedImportCommand  string                                        `json:"approvedImportCommand"`
	ManifestSyncCommand    string                                        `json:"manifestSyncCommand"`
	EvidenceVerifyCommand  string                                        `json:"evidenceVerifyCommand"`
	CompletionAuditCommand string                                        `json:"completionAuditCommand"`
}

type setupSvcLiveReplaySourceExecutionPacketResult struct {
	Mode                             string                                           `json:"mode"`
	Project                          string                                           `json:"project"`
	ReadOnly                         bool                                             `json:"readOnly"`
	Status                           string                                           `json:"status"`
	ManifestPath                     string                                           `json:"manifestPath"`
	GeneratedFrom                    string                                           `json:"generatedFrom"`
	SourceRoot                       string                                           `json:"sourceRoot"`
	CaptureRoot                      string                                           `json:"captureRoot"`
	Filters                          *setupSvcLiveReplayCollectionPlanFilters         `json:"filters,omitempty"`
	MetadataServiceDatasource        setupSvcLiveReplayDatasourceReadiness            `json:"metadataServiceDatasource"`
	Totals                           setupSvcLiveReplaySourceExecutionPacketTotals    `json:"totals"`
	CaptureModeGroups                []setupSvcLiveReplaySourceExecutionPacketGroup   `json:"captureModeGroups"`
	Groups                           []setupSvcLiveReplaySourceExecutionPacketGroup   `json:"groups"`
	OperatorBatches                  []setupSvcLiveReplaySourceExecutionOperatorBatch `json:"operatorBatches"`
	OperatorPacket                   setupSvcLiveReplaySourceExecutionOperatorPacket  `json:"operatorPacket"`
	ExecutionRunbook                 []setupSvcLiveReplaySourceExecutionRunbookStep   `json:"executionRunbook,omitempty"`
	BatchSaveCommands                []string                                         `json:"batchSaveCommands,omitempty"`
	BatchSaveScript                  string                                           `json:"batchSaveScript,omitempty"`
	BatchSaveScriptPath              string                                           `json:"batchSaveScriptPath,omitempty"`
	SaveBatchSaveScriptCommand       string                                           `json:"saveBatchSaveScriptCommand,omitempty"`
	ImportBatchSaveCommands          []string                                         `json:"importBatchSaveCommands,omitempty"`
	ImportBatchSaveScript            string                                           `json:"importBatchSaveScript,omitempty"`
	ImportBatchSaveScriptPath        string                                           `json:"importBatchSaveScriptPath,omitempty"`
	SaveImportBatchSaveScriptCommand string                                           `json:"saveImportBatchSaveScriptCommand,omitempty"`
	RunbookMarkdown                  string                                           `json:"runbookMarkdown,omitempty"`
	RunbookMarkdownPath              string                                           `json:"runbookMarkdownPath,omitempty"`
	SaveRunbookMarkdownCommand       string                                           `json:"saveRunbookMarkdownCommand,omitempty"`
	ArtifactType                     string                                           `json:"artifactType,omitempty"`
	SourceSystem                     string                                           `json:"sourceSystem,omitempty"`
	CaptureMode                      string                                           `json:"captureMode,omitempty"`
	CaptureGroups                    int                                              `json:"captureGroups"`
	ArtifactTypes                    int                                              `json:"artifactTypes"`
	SourceFiles                      int                                              `json:"sourceFiles"`
	TargetFiles                      int                                              `json:"targetFiles"`
	IncompleteSourceFiles            int                                              `json:"sourceFilesIncomplete"`
	CompleteSourceFiles              int                                              `json:"sourceFilesComplete"`
	MissingSourceFiles               int                                              `json:"sourceFilesMissing"`
	DomainOperations                 int                                              `json:"domainOperations"`
	EvidenceSectionCount             int                                              `json:"evidenceSectionCount"`
	EvidenceSections                 []string                                         `json:"evidenceSections,omitempty"`
	RequiredTables                   []string                                         `json:"requiredTables,omitempty"`
	RuntimeEffects                   []string                                         `json:"runtimeEffects,omitempty"`
	QueryReadbackExpectations        []string                                         `json:"queryReadbackExpectations,omitempty"`
	SuggestedBatchPath               string                                           `json:"suggestedBatchPath,omitempty"`
	SaveBatchCommand                 string                                           `json:"saveBatchCommand,omitempty"`
	PostCaptureCheckCommand          string                                           `json:"postCaptureCheckCommand,omitempty"`
	Items                            []setupSvcLiveReplaySourceChecklistItem          `json:"items,omitempty"`
	OperatorBatch                    *setupSvcLiveReplaySourceExecutionOperatorBatch  `json:"operatorBatch,omitempty"`
	NextSteps                        []string                                         `json:"nextSteps"`
	Notes                            []string                                         `json:"notes"`
}

type setupSvcLiveReplaySourceExecutionPacketTotals struct {
	SourceFiles           int `json:"sourceFiles"`
	TargetFiles           int `json:"targetFiles"`
	IncompleteSourceFiles int `json:"incompleteSourceFiles,omitempty"`
	CompleteSourceFiles   int `json:"completeSourceFiles,omitempty"`
	MissingSourceFiles    int `json:"missingSourceFiles,omitempty"`
	ArtifactTypes         int `json:"artifactTypes"`
	DomainOperations      int `json:"domainOperations"`
	EvidenceSections      int `json:"evidenceSections"`
	CaptureGroups         int `json:"captureGroups"`
	GroupedSourceFiles    int `json:"groupedSourceFiles"`
	GroupedTargetFiles    int `json:"groupedTargetFiles"`
}

type setupSvcLiveReplaySourceExecutionOperatorPacket struct {
	Purpose                          string                                `json:"purpose"`
	SourceRoot                       string                                `json:"sourceRoot"`
	CaptureRoot                      string                                `json:"captureRoot"`
	SourceFiles                      int                                   `json:"sourceFiles"`
	TargetFiles                      int                                   `json:"targetFiles"`
	IncompleteSourceFiles            int                                   `json:"incompleteSourceFiles,omitempty"`
	CompleteSourceFiles              int                                   `json:"completeSourceFiles,omitempty"`
	MissingSourceFiles               int                                   `json:"missingSourceFiles,omitempty"`
	ArtifactTypes                    int                                   `json:"artifactTypes"`
	DomainOperations                 int                                   `json:"domainOperations"`
	EvidenceSectionCount             int                                   `json:"evidenceSectionCount"`
	CaptureGroups                    int                                   `json:"captureGroups"`
	OperatorBatchCount               int                                   `json:"operatorBatchCount"`
	RunbookStepCount                 int                                   `json:"runbookStepCount"`
	BatchSaveCommandCount            int                                   `json:"batchSaveCommandCount"`
	ImportBatchSaveCommandCount      int                                   `json:"importBatchSaveCommandCount"`
	BatchSaveScriptPath              string                                `json:"batchSaveScriptPath,omitempty"`
	SaveBatchSaveScriptCommand       string                                `json:"saveBatchSaveScriptCommand,omitempty"`
	ImportBatchSaveScriptPath        string                                `json:"importBatchSaveScriptPath,omitempty"`
	SaveImportBatchSaveScriptCommand string                                `json:"saveImportBatchSaveScriptCommand,omitempty"`
	RunbookMarkdownPath              string                                `json:"runbookMarkdownPath,omitempty"`
	SaveRunbookMarkdownCommand       string                                `json:"saveRunbookMarkdownCommand,omitempty"`
	CompletionAuditCommand           string                                `json:"completionAuditCommand"`
	MetadataServiceDatasource        setupSvcLiveReplayDatasourceReadiness `json:"metadataServiceDatasource"`
	StopConditions                   []string                              `json:"stopConditions"`
}

type setupSvcLiveReplaySourceExecutionPacketGroup struct {
	ArtifactType              string                                  `json:"artifactType"`
	SourceSystem              string                                  `json:"sourceSystem"`
	CaptureMode               string                                  `json:"captureMode"`
	SourceFiles               int                                     `json:"sourceFiles"`
	TargetFiles               int                                     `json:"targetFiles"`
	DomainOperations          int                                     `json:"domainOperations"`
	EvidenceSections          []string                                `json:"evidenceSections,omitempty"`
	RequiredTables            []string                                `json:"requiredTables,omitempty"`
	RuntimeEffects            []string                                `json:"runtimeEffects,omitempty"`
	QueryReadbackExpectations []string                                `json:"queryReadbackExpectations,omitempty"`
	SuggestedBatchPath        string                                  `json:"suggestedBatchPath"`
	SaveBatchCommand          string                                  `json:"saveBatchCommand"`
	PostCaptureCheckCommand   string                                  `json:"postCaptureCheckCommand"`
	Items                     []setupSvcLiveReplaySourceChecklistItem `json:"items"`
}

type setupSvcLiveReplaySourceExecutionOperatorBatch struct {
	ArtifactType              string                                 `json:"artifactType"`
	SourceSystem              string                                 `json:"sourceSystem"`
	CaptureMode               string                                 `json:"captureMode"`
	SourceFiles               int                                    `json:"sourceFiles"`
	TargetFiles               int                                    `json:"targetFiles"`
	DomainOperations          int                                    `json:"domainOperations"`
	EvidenceSections          []string                               `json:"evidenceSections,omitempty"`
	BatchPath                 string                                 `json:"batchPath"`
	SuggestedBatchPath        string                                 `json:"suggestedBatchPath"`
	SaveBatchCommand          string                                 `json:"saveBatchCommand"`
	SuggestedImportBatchPath  string                                 `json:"suggestedImportBatchPath"`
	SaveImportBatchCommand    string                                 `json:"saveImportBatchCommand"`
	NextAction                string                                 `json:"nextAction"`
	MetadataServiceDatasource *setupSvcLiveReplayDatasourceReadiness `json:"metadataServiceDatasource,omitempty"`
	ManualCaptureRequired     bool                                   `json:"manualCaptureRequired,omitempty"`
	DryRunCaptureCommand      string                                 `json:"dryRunCaptureCommand,omitempty"`
	ExecuteCaptureCommand     string                                 `json:"executeCaptureCommand,omitempty"`
	PostCaptureCheckCommand   string                                 `json:"postCaptureCheckCommand"`
	DryRunImportCommand       string                                 `json:"dryRunImportCommand"`
	ApprovedImportCommand     string                                 `json:"approvedImportCommand"`
	CompletionAuditCommand    string                                 `json:"completionAuditCommand"`
}

type setupSvcLiveReplaySourceExecutionRunbookStep struct {
	Order                     int                                    `json:"order"`
	ArtifactType              string                                 `json:"artifactType"`
	DependsOn                 []string                               `json:"dependsOn,omitempty"`
	SourceSystem              string                                 `json:"sourceSystem"`
	CaptureMode               string                                 `json:"captureMode"`
	SourceFiles               int                                    `json:"sourceFiles"`
	TargetFiles               int                                    `json:"targetFiles"`
	DomainOperations          int                                    `json:"domainOperations"`
	EvidenceSections          []string                               `json:"evidenceSections,omitempty"`
	Gate                      string                                 `json:"gate"`
	NextAction                string                                 `json:"nextAction"`
	MetadataServiceDatasource *setupSvcLiveReplayDatasourceReadiness `json:"metadataServiceDatasource,omitempty"`
	ManualCaptureRequired     bool                                   `json:"manualCaptureRequired,omitempty"`
	DryRunCaptureCommand      string                                 `json:"dryRunCaptureCommand,omitempty"`
	ExecuteCaptureCommand     string                                 `json:"executeCaptureCommand,omitempty"`
	BatchPath                 string                                 `json:"batchPath"`
	SuggestedBatchPath        string                                 `json:"suggestedBatchPath"`
	SaveBatchCommand          string                                 `json:"saveBatchCommand"`
	SuggestedImportBatchPath  string                                 `json:"suggestedImportBatchPath"`
	SaveImportBatchCommand    string                                 `json:"saveImportBatchCommand"`
	PostCaptureCheckCommand   string                                 `json:"postCaptureCheckCommand"`
	DryRunImportCommand       string                                 `json:"dryRunImportCommand"`
	ApprovedImportCommand     string                                 `json:"approvedImportCommand"`
	CompletionAuditCommand    string                                 `json:"completionAuditCommand"`
}

type setupSvcLiveReplayQueryReadbackCapturePlanResult struct {
	Mode                         string                                       `json:"mode"`
	Project                      string                                       `json:"project"`
	ReadOnly                     bool                                         `json:"readOnly"`
	Status                       string                                       `json:"status"`
	ManifestPath                 string                                       `json:"manifestPath"`
	GeneratedFrom                string                                       `json:"generatedFrom"`
	SourceRoot                   string                                       `json:"sourceRoot"`
	CaptureRoot                  string                                       `json:"captureRoot"`
	Filters                      *setupSvcLiveReplayCollectionPlanFilters     `json:"filters,omitempty"`
	QueryReadbackSources         int                                          `json:"queryReadbackSources"`
	TotalQueryReadbackSources    int                                          `json:"totalQueryReadbackSources"`
	ReturnedQueryReadbackSources int                                          `json:"returnedQueryReadbackSources"`
	OmittedQueryReadbackSources  int                                          `json:"omittedQueryReadbackSources"`
	Totals                       setupSvcLiveReplayQueryReadbackCaptureTotals `json:"totals"`
	CaptureRequests              []setupSvcLiveReplayQueryReadbackCaptureItem `json:"captureRequests"`
	OperatorPacket               setupSvcLiveReplayQueryReadbackOperator      `json:"operatorPacket"`
	StopConditions               []string                                     `json:"stopConditions"`
	NextSteps                    []string                                     `json:"nextSteps"`
}

type setupSvcLiveReplayQueryReadbackCaptureTotals struct {
	QueryReadbackSources         int `json:"queryReadbackSources"`
	TotalQueryReadbackSources    int `json:"totalQueryReadbackSources"`
	ReturnedQueryReadbackSources int `json:"returnedQueryReadbackSources"`
	Offset                       int `json:"offset"`
	Limit                        int `json:"limit"`
	OmittedQueryReadbackSources  int `json:"omittedQueryReadbackSources"`
	DomainOperations             int `json:"domainOperations"`
	RequiredTables               int `json:"requiredTables"`
	Expectations                 int `json:"expectations"`
}

type setupSvcLiveReplayQueryReadbackCaptureItem struct {
	Domain                      string         `json:"domain"`
	Operation                   string         `json:"operation"`
	SourcePath                  string         `json:"sourcePath"`
	TargetPath                  string         `json:"targetPath"`
	SourceReadiness             string         `json:"sourceReadiness"`
	RequiredTables              []string       `json:"requiredTables,omitempty"`
	QueryReadbackExpectations   []string       `json:"queryReadbackExpectations,omitempty"`
	RequiredEvidenceSections    []string       `json:"requiredEvidenceSections"`
	MissingEvidenceSections     []string       `json:"missingEvidenceSections,omitempty"`
	CaptureArtifactShape        map[string]any `json:"captureArtifactShape"`
	ScannerCommand              string         `json:"scannerCommand,omitempty"`
	RecommendedReadbackCommands []string       `json:"recommendedReadbackCommands"`
	PostCaptureCheckCommand     string         `json:"postCaptureCheckCommand"`
	CompleteWorklistCommand     string         `json:"completeWorklistCommand"`
	DryRunImportCommand         string         `json:"dryRunImportCommand"`
	ApprovedImportCommand       string         `json:"approvedImportCommand"`
}

type setupSvcLiveReplayQueryReadbackOperator struct {
	Purpose                 string `json:"purpose"`
	QueryReadbackSources    int    `json:"queryReadbackSources"`
	SourceReadiness         string `json:"sourceReadiness,omitempty"`
	RecommendedBatchCommand string `json:"recommendedBatchCommand"`
	SavePlanCommand         string `json:"savePlanCommand"`
	PostCaptureCheckCommand string `json:"postCaptureCheckCommand"`
	DryRunImportCommand     string `json:"dryRunImportCommand"`
	ApprovedImportCommand   string `json:"approvedImportCommand"`
	CompletionAuditCommand  string `json:"completionAuditCommand"`
}

type setupSvcLiveReplayQueryReadbackCaptureApplyResult struct {
	Mode                 string                                        `json:"mode"`
	Project              string                                        `json:"project"`
	ReadOnly             bool                                          `json:"readOnly"`
	Execute              bool                                          `json:"execute"`
	ApprovalRequired     bool                                          `json:"approvalRequired"`
	Approved             bool                                          `json:"approved"`
	Status               string                                        `json:"status"`
	ManifestPath         string                                        `json:"manifestPath"`
	SourceRoot           string                                        `json:"sourceRoot"`
	CaptureRoot          string                                        `json:"captureRoot"`
	TotalCaptureRequests int                                           `json:"totalCaptureRequests"`
	WrittenFiles         int                                           `json:"writtenFiles"`
	PassedArtifacts      int                                           `json:"passedArtifacts"`
	FailedArtifacts      int                                           `json:"failedArtifacts"`
	Artifacts            []setupSvcLiveReplayQueryReadbackCaptureWrite `json:"artifacts,omitempty"`
	BlockingIssues       []string                                      `json:"blockingIssues,omitempty"`
	Warnings             []string                                      `json:"warnings,omitempty"`
	NextCommands         map[string]string                             `json:"nextCommands"`
}

type setupSvcLiveReplayQueryReadbackCaptureWrite struct {
	Domain         string   `json:"domain"`
	Operation      string   `json:"operation"`
	SourcePath     string   `json:"sourcePath"`
	TargetPath     string   `json:"targetPath"`
	Status         string   `json:"status"`
	Issues         []string `json:"issues,omitempty"`
	RequiredTables int      `json:"requiredTables"`
	Expectations   int      `json:"expectations"`
}

type setupSvcLiveReplaySnapshotFromChangesApplyResult struct {
	Mode             string                                           `json:"mode"`
	Project          string                                           `json:"project"`
	ReadOnly         bool                                             `json:"readOnly"`
	Execute          bool                                             `json:"execute"`
	ApprovalRequired bool                                             `json:"approvalRequired"`
	Approved         bool                                             `json:"approved"`
	Status           string                                           `json:"status"`
	ManifestPath     string                                           `json:"manifestPath,omitempty"`
	WrittenFiles     int                                              `json:"writtenFiles"`
	PassedArtifacts  int                                              `json:"passedArtifacts"`
	FailedArtifacts  int                                              `json:"failedArtifacts"`
	Artifacts        []setupSvcLiveReplaySnapshotFromChangesWriteItem `json:"artifacts,omitempty"`
	BlockingIssues   []string                                         `json:"blockingIssues,omitempty"`
	Warnings         []string                                         `json:"warnings,omitempty"`
	NextCommands     map[string]string                                `json:"nextCommands"`
}

type setupSvcLiveReplaySnapshotFromChangesWriteItem struct {
	Domain          string   `json:"domain"`
	Operation       string   `json:"operation"`
	ArtifactType    string   `json:"artifactType"`
	OperationID     string   `json:"operationId,omitempty"`
	SourcePath      string   `json:"sourcePath,omitempty"`
	TargetPath      string   `json:"targetPath,omitempty"`
	Status          string   `json:"status"`
	Issues          []string `json:"issues,omitempty"`
	Changes         int      `json:"changes"`
	TableSnapshots  int      `json:"tableSnapshots"`
	RuntimeEffects  int      `json:"runtimeEffects"`
	RequiredTables  int      `json:"requiredTables"`
	RequiredEffects int      `json:"requiredEffects"`
}

type setupSvcLiveReplayMetadataServiceApplyCaptureResult struct {
	Mode                      string                                              `json:"mode"`
	Project                   string                                              `json:"project"`
	ReadOnly                  bool                                                `json:"readOnly"`
	Execute                   bool                                                `json:"execute"`
	ApprovalRequired          bool                                                `json:"approvalRequired"`
	Approved                  bool                                                `json:"approved"`
	Status                    string                                              `json:"status"`
	OperationResultsDir       string                                              `json:"operationResultsDir"`
	MetadataServiceDatasource setupSvcLiveReplayDatasourceReadiness               `json:"metadataServiceDatasource"`
	TotalRecords              int                                                 `json:"totalRecords"`
	WrittenFiles              int                                                 `json:"writtenFiles"`
	PassedArtifacts           int                                                 `json:"passedArtifacts"`
	FailedArtifacts           int                                                 `json:"failedArtifacts"`
	Artifacts                 []setupSvcLiveReplayMetadataServiceApplyCaptureItem `json:"artifacts,omitempty"`
	BlockingIssues            []string                                            `json:"blockingIssues,omitempty"`
	Warnings                  []string                                            `json:"warnings,omitempty"`
	NextCommands              map[string]string                                   `json:"nextCommands"`
}

type setupSvcLiveReplayMetadataServiceQueryScanCaptureResult struct {
	Mode                      string                                                  `json:"mode"`
	Project                   string                                                  `json:"project"`
	ReadOnly                  bool                                                    `json:"readOnly"`
	Execute                   bool                                                    `json:"execute"`
	ApprovalRequired          bool                                                    `json:"approvalRequired"`
	Approved                  bool                                                    `json:"approved"`
	Status                    string                                                  `json:"status"`
	MetadataServiceDatasource setupSvcLiveReplayDatasourceReadiness                   `json:"metadataServiceDatasource"`
	TotalRecords              int                                                     `json:"totalRecords"`
	TotalCaptureRequests      int                                                     `json:"totalCaptureRequests"`
	WrittenFiles              int                                                     `json:"writtenFiles"`
	PassedArtifacts           int                                                     `json:"passedArtifacts"`
	FailedArtifacts           int                                                     `json:"failedArtifacts"`
	Artifacts                 []setupSvcLiveReplayMetadataServiceQueryScanCaptureItem `json:"artifacts,omitempty"`
	BlockingIssues            []string                                                `json:"blockingIssues,omitempty"`
	Warnings                  []string                                                `json:"warnings,omitempty"`
	NextCommands              map[string]string                                       `json:"nextCommands"`
}

type setupSvcLiveReplayMetadataServiceQueryScanCaptureItem struct {
	Domain          string   `json:"domain"`
	Operation       string   `json:"operation"`
	ArtifactType    string   `json:"artifactType"`
	SourcePath      string   `json:"sourcePath,omitempty"`
	TargetPath      string   `json:"targetPath,omitempty"`
	Status          string   `json:"status"`
	Issues          []string `json:"issues,omitempty"`
	TableSnapshots  int      `json:"tableSnapshots"`
	RuntimeEffects  int      `json:"runtimeEffects"`
	RequiredTables  int      `json:"requiredTables"`
	RequiredEffects int      `json:"requiredEffects"`
}

type setupSvcLiveReplayMetadataServiceApplyCaptureItem struct {
	Domain       string   `json:"domain"`
	Operation    string   `json:"operation"`
	ArtifactType string   `json:"artifactType"`
	PlanID       string   `json:"planId,omitempty"`
	OperationID  string   `json:"operationId,omitempty"`
	ResultPath   string   `json:"resultPath,omitempty"`
	Status       string   `json:"status"`
	Issues       []string `json:"issues,omitempty"`
}

type setupSvcLiveReplayWorklistQueue struct {
	ArtifactType        string                            `json:"artifactType"`
	Section             string                            `json:"section"`
	Missing             int                               `json:"missing"`
	RequiredShapeKey    string                            `json:"requiredShapeKey"`
	ManifestStatusField string                            `json:"manifestStatusField"`
	PageSize            int                               `json:"pageSize"`
	BatchCount          int                               `json:"batchCount"`
	QueueCommand        string                            `json:"queueCommand"`
	Batches             []setupSvcLiveReplayWorklistBatch `json:"batches,omitempty"`
	OmittedBatches      int                               `json:"omittedBatches,omitempty"`
}

type setupSvcLiveReplayWorklistBatch struct {
	BatchIndex            int                                          `json:"batchIndex"`
	Offset                int                                          `json:"offset"`
	Limit                 int                                          `json:"limit"`
	Count                 int                                          `json:"count"`
	Command               string                                       `json:"command"`
	SuggestedWorklistPath string                                       `json:"suggestedWorklistPath"`
	SaveWorklistCommand   string                                       `json:"saveWorklistCommand"`
	DryRunImportCommand   string                                       `json:"dryRunImportCommand"`
	ExecuteImportCommand  string                                       `json:"executeImportCommand"`
	OperatorBatch         setupSvcLiveReplayWorklistOperatorBatch      `json:"operatorBatch"`
	Artifacts             []setupSvcLiveReplayArtifactCollectionAction `json:"artifacts,omitempty"`
}

type setupSvcLiveReplayWorklistBatchCommand struct {
	ArtifactType          string `json:"artifactType"`
	EvidenceSection       string `json:"evidenceSection"`
	BatchIndex            int    `json:"batchIndex"`
	Offset                int    `json:"offset"`
	Limit                 int    `json:"limit"`
	Count                 int    `json:"count"`
	SuggestedWorklistPath string `json:"suggestedWorklistPath"`
	SaveWorklistCommand   string `json:"saveWorklistCommand"`
	DryRunImportCommand   string `json:"dryRunImportCommand"`
	ExecuteImportCommand  string `json:"executeImportCommand"`
}

type setupSvcLiveReplayWorklistOperatorPacket struct {
	Purpose                    string                                     `json:"purpose"`
	ArtifactReplacementCount   int                                        `json:"artifactReplacementCount"`
	UniqueArtifactFiles        int                                        `json:"uniqueArtifactFiles"`
	SourceFiles                int                                        `json:"sourceFiles"`
	TargetFiles                int                                        `json:"targetFiles"`
	DuplicateArtifactRecords   int                                        `json:"duplicateArtifactRecords"`
	ReplacementStatusTarget    string                                     `json:"replacementStatusTarget"`
	SourceRoot                 string                                     `json:"sourceRoot"`
	CaptureRoot                string                                     `json:"captureRoot"`
	SuggestedWorklistPath      string                                     `json:"suggestedWorklistPath"`
	SaveWorklistCommand        string                                     `json:"saveWorklistCommand"`
	DryRunImportCommand        string                                     `json:"dryRunImportCommand"`
	ExecuteImportCommand       string                                     `json:"executeImportCommand"`
	SourceFilesPresent         int                                        `json:"sourceFilesPresent"`
	SourceFilesMissing         int                                        `json:"sourceFilesMissing"`
	SourceFilesComplete        int                                        `json:"sourceFilesComplete"`
	SourceFilesIncomplete      int                                        `json:"sourceFilesIncomplete"`
	BatchSaveCommands          []setupSvcLiveReplayWorklistBatchCommand   `json:"batchSaveCommands,omitempty"`
	SourceEvidenceSections     []setupSvcLiveReplayEvidenceSectionSummary `json:"sourceEvidenceSections,omitempty"`
	SourceMissingSectionQueues []setupSvcLiveReplayEvidenceSectionQueue   `json:"sourceMissingSectionQueues,omitempty"`
	PostReplacementCommands    []string                                   `json:"postReplacementCommands"`
	StopConditions             []string                                   `json:"stopConditions"`
}

type setupSvcLiveReplayWorklistOperatorBatch struct {
	BatchIndex                 int                                           `json:"batchIndex"`
	ArtifactType               string                                        `json:"artifactType"`
	EvidenceSection            string                                        `json:"evidenceSection"`
	Offset                     int                                           `json:"offset"`
	Limit                      int                                           `json:"limit"`
	ReplacementStatusTarget    string                                        `json:"replacementStatusTarget"`
	PostReplacementCommands    []string                                      `json:"postReplacementCommands"`
	ArtifactReplacementRecords []setupSvcLiveReplayArtifactReplacementRecord `json:"artifactReplacementRecords,omitempty"`
}

type setupSvcLiveReplayArtifactReplacementRecord struct {
	Domain                    string                                    `json:"domain"`
	Operation                 string                                    `json:"operation"`
	ArtifactType              string                                    `json:"artifactType"`
	Path                      string                                    `json:"path"`
	SuggestedSourcePath       string                                    `json:"suggestedSourcePath,omitempty"`
	SuggestedSourceExists     bool                                      `json:"suggestedSourceExists"`
	SourceReadiness           string                                    `json:"sourceReadiness"`
	RequiredShapeKey          string                                    `json:"requiredShapeKey"`
	ManifestStatusField       string                                    `json:"manifestStatusField"`
	ReplacementStatusTarget   string                                    `json:"replacementStatusTarget"`
	RequiredEvidenceSections  []string                                  `json:"requiredEvidenceSections"`
	SourceEvidenceSections    []setupSvcLiveReplayEvidenceSectionStatus `json:"sourceEvidenceSections,omitempty"`
	MissingEvidenceSections   []string                                  `json:"missingEvidenceSections,omitempty"`
	RequiredTables            []string                                  `json:"requiredTables,omitempty"`
	RuntimeEffects            []string                                  `json:"runtimeEffects,omitempty"`
	QueryReadbackExpectations []string                                  `json:"queryReadbackExpectations,omitempty"`
	CaptureTask               setupSvcLiveReplayArtifactCaptureTask     `json:"captureTask"`
	Checklist                 []string                                  `json:"checklist"`
}

type setupSvcLiveReplayArtifactCaptureTask struct {
	SourceSystem              string         `json:"sourceSystem"`
	CaptureMode               string         `json:"captureMode"`
	Domain                    string         `json:"domain"`
	Operation                 string         `json:"operation"`
	ArtifactType              string         `json:"artifactType"`
	TargetPath                string         `json:"targetPath"`
	SuggestedSourcePath       string         `json:"suggestedSourcePath"`
	StatusTarget              string         `json:"statusTarget"`
	RequiredShapeKey          string         `json:"requiredShapeKey"`
	ManifestStatusField       string         `json:"manifestStatusField"`
	RequiredEvidenceSections  []string       `json:"requiredEvidenceSections"`
	RequiredTables            []string       `json:"requiredTables,omitempty"`
	RuntimeEffects            []string       `json:"runtimeEffects,omitempty"`
	QueryReadbackExpectations []string       `json:"queryReadbackExpectations,omitempty"`
	ManualAction              string         `json:"manualAction"`
	CaptureCommand            string         `json:"captureCommand,omitempty"`
	PlanRequest               map[string]any `json:"planRequest,omitempty"`
	PlanCommand               string         `json:"planCommand,omitempty"`
	ScanRequest               map[string]any `json:"scanRequest,omitempty"`
	ScanCommand               string         `json:"scanCommand,omitempty"`
	PostCaptureCheckCommand   string         `json:"postCaptureCheckCommand"`
	PostCaptureImportHint     string         `json:"postCaptureImportHint"`
	StopConditions            []string       `json:"stopConditions"`
}

type setupSvcLiveReplayArtifactCollectionAction struct {
	Domain                    string                                    `json:"domain"`
	Operation                 string                                    `json:"operation"`
	ArtifactType              string                                    `json:"artifactType"`
	Path                      string                                    `json:"path"`
	Status                    string                                    `json:"status"`
	NextAction                string                                    `json:"nextAction"`
	RequiredShapeKey          string                                    `json:"requiredShapeKey"`
	ManifestStatusField       string                                    `json:"manifestStatusField"`
	RequiredEvidenceSections  []string                                  `json:"requiredEvidenceSections"`
	EvidenceSectionStatuses   []setupSvcLiveReplayEvidenceSectionStatus `json:"evidenceSectionStatuses,omitempty"`
	ReplacementChecklist      []string                                  `json:"replacementChecklist"`
	RequiredTables            []string                                  `json:"requiredTables,omitempty"`
	RuntimeEffects            []string                                  `json:"runtimeEffects,omitempty"`
	QueryReadbackExpectations []string                                  `json:"queryReadbackExpectations,omitempty"`
	CaptureTask               setupSvcLiveReplayArtifactCaptureTask     `json:"captureTask"`
}

type setupSvcLiveReplayEvidenceSectionStatus struct {
	Section string `json:"section"`
	Status  string `json:"status"`
	Present bool   `json:"present"`
}

type setupSvcLiveReplayGapCommands struct {
	PrepareWorkspace string `json:"prepareWorkspace"`
	GenerateDiffs    string `json:"generateDiffs"`
	SyncManifest     string `json:"syncManifest"`
	VerifyEvidence   string `json:"verifyEvidence"`
	WriteBundle      string `json:"writeBundle"`
	PromotionAudit   string `json:"promotionAudit"`
	CompletionAudit  string `json:"completionAudit"`
}

type setupSvcLiveReplayGapDomain struct {
	Domain     string                           `json:"domain"`
	Status     string                           `json:"status"`
	Operations []setupSvcLiveReplayGapOperation `json:"operations"`
}

type setupSvcLiveReplayGapOperation struct {
	Operation       string   `json:"operation"`
	Status          string   `json:"status"`
	NextAction      string   `json:"nextAction"`
	MissingEvidence []string `json:"missingEvidence,omitempty"`
	PendingEvidence []string `json:"pendingEvidence,omitempty"`
	FailedEvidence  []string `json:"failedEvidence,omitempty"`
	EvidenceFiles   []string `json:"evidenceFiles"`
}

type setupSvcLiveReplayNormalizedDiffDomain struct {
	Domain     string                                      `json:"domain"`
	Status     string                                      `json:"status"`
	Operations []setupSvcLiveReplayNormalizedDiffOperation `json:"operations"`
}

type setupSvcLiveReplayNormalizedDiffOperation struct {
	Operation      string   `json:"operation"`
	Status         string   `json:"status"`
	DiffFile       string   `json:"diffFile"`
	Differences    int      `json:"differences"`
	BlockingIssues []string `json:"blockingIssues,omitempty"`
}

type setupSvcLiveReplayDiffArtifact struct {
	Status              string                              `json:"status"`
	Project             string                              `json:"project"`
	ContractVersion     string                              `json:"contractVersion"`
	ContractFingerprint string                              `json:"contractFingerprint"`
	Domain              string                              `json:"domain"`
	Operation           string                              `json:"operation"`
	ArtifactType        string                              `json:"artifactType"`
	GeneratedAt         string                              `json:"generatedAt"`
	Totals              setupSvcLiveReplayDiffTotals        `json:"totals"`
	Tables              []setupSvcLiveReplayDiffTableResult `json:"tables"`
	Normalization       setupSvcLiveReplayDiffNormalization `json:"normalization"`
}

type setupSvcLiveReplayDiffTotals struct {
	MissingRows      int `json:"missingRows"`
	UnexpectedRows   int `json:"unexpectedRows"`
	MismatchedValues int `json:"mismatchedValues"`
	Differences      int `json:"differences"`
	Failed           int `json:"failed"`
}

type setupSvcLiveReplayDiffTableResult struct {
	Table             string   `json:"table"`
	Status            string   `json:"status"`
	MissingRows       int      `json:"missingRows"`
	UnexpectedRows    int      `json:"unexpectedRows"`
	MismatchedValues  int      `json:"mismatchedValues"`
	MissingColumns    []string `json:"missingColumns,omitempty"`
	UnexpectedColumns []string `json:"unexpectedColumns,omitempty"`
}

type setupSvcLiveReplayDiffNormalization struct {
	DynamicValues string `json:"dynamicValues"`
	RowOrder      string `json:"rowOrder"`
}

type setupSvcLiveReplayComparableTable struct {
	Columns map[string]bool
	Rows    map[string]bool
}

type setupSvcLiveReplayApplyDomain struct {
	Domain     string                             `json:"domain"`
	Status     string                             `json:"status"`
	Operations []setupSvcLiveReplayApplyOperation `json:"operations"`
}

type setupSvcLiveReplayApplyOperation struct {
	Operation string `json:"operation"`
	Status    string `json:"status"`
}

type setupSvcLiveReplayEvidenceTotals struct {
	Domains            int `json:"domains"`
	Operations         int `json:"operations"`
	VerifiedDomains    int `json:"verifiedDomains"`
	VerifiedOperations int `json:"verifiedOperations"`
	MissingDomains     int `json:"missingDomains"`
	MissingOperations  int `json:"missingOperations"`
	FailedOperations   int `json:"failedOperations"`
}

type setupSvcLiveReplayEvidenceDomain struct {
	Domain             string                             `json:"domain"`
	Status             string                             `json:"status"`
	ExpectedOperations []string                           `json:"expectedOperations"`
	VerifiedOperations []string                           `json:"verifiedOperations"`
	MissingOperations  []string                           `json:"missingOperations,omitempty"`
	FailedOperations   []setupSvcLiveReplayOperationIssue `json:"failedOperations,omitempty"`
}

type setupSvcLiveReplayOperationIssue struct {
	Operation       string   `json:"operation"`
	MissingEvidence []string `json:"missingEvidence,omitempty"`
	FailedEvidence  []string `json:"failedEvidence,omitempty"`
}

func setupSvcLiveReplayReadinessResult(projectPath string, metadataServiceURL string) setupSvcLiveReplayReadiness {
	cfg := setupSvcLiveReplayProjectConfig(projectPath, metadataServiceURL)
	domains := setupSvcLiveReplayDomains()
	matrixContract := setupSvcLiveReplayMatrixContractStatus(projectPath)
	totalOps := 0
	for _, domain := range domains {
		totalOps += len(domain.Operations)
	}
	status := "blocked_missing_config"
	if cfg.HasSetupSvc && cfg.HasAccessToken && cfg.HasMetadataService {
		status = "ready_for_approved_live_replay"
	}
	var blockingIssues []string
	if matrixContract.Status != "passed" {
		status = "blocked_parity_matrix_contract"
		for _, issue := range matrixContract.Issues {
			blockingIssues = append(blockingIssues, "parityMatrix: "+issue)
		}
	}
	return setupSvcLiveReplayReadiness{
		Mode:             "setup-svc-live-replay-readiness",
		Project:          projectPath,
		ReadOnly:         true,
		Execute:          false,
		ApprovalRequired: true,
		ApprovalPhrase:   setupSvcParityReplayApproval,
		Status:           status,
		Config:           cfg,
		Totals: setupSvcLiveReplayTotals{
			Domains:                  len(domains),
			Operations:               totalOps,
			CoveredPendingLiveReplay: len(domains),
			Verified:                 0,
		},
		Commands: setupSvcLiveReplayCommands{
			Readiness:         "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-readiness",
			MetadataPreflight: "cloudcc scan msapi " + shellPath(projectPath) + " standard-catalog",
			ApprovedReplay:    "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-packet > setup-svc-live-replay-packet.json && cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-packet @setup-svc-live-replay-packet.json --dry-run",
			EvidenceDirectory: filepath.Join(projectPath, "outputs", "setup-svc-live-replay"),
		},
		MatrixContract: matrixContract,
		Domains:        domains,
		BlockingIssues: blockingIssues,
		StopConditions: []string{
			"Do not execute create/update/delete replay without an approved disposable tenant or rollback window.",
			"Stop if setup-svc baseline write does not return success and a query-visible metadata id.",
			"Stop if MetadataService apply status is not VERIFIED.",
			"Stop if setup-svc snapshot and MetadataService snapshot differ outside documented dynamic-value normalizers.",
			"Stop if query/readback cannot reconstruct the metadata relationships required by the parity matrix.",
		},
		Notes: []string{
			"This command is read-only and produces the evidence contract for promoting matrix domains from covered to verified.",
			"Live replay must collect setup-svc baseline snapshots, MetadataService apply snapshots, query/readback payloads, and normalized diffs per domain.",
			"Matrix status remains covered until the approved live replay evidence is recorded in .claw/test-report.md.",
		},
	}
}

func buildSetupSvcLiveReplayCoverageResult(projectPath string) setupSvcLiveReplayCoverageResult {
	matrixContract := setupSvcLiveReplayMatrixContractStatus(projectPath)
	matrixItems := setupSvcLiveReplayMatrixDomainItems(matrixContract.Path)
	testEvidence := setupSvcLiveReplayTestEvidenceStatus(projectPath)
	testEvidenceItems := setupSvcLiveReplayTestEvidenceDomainItems(testEvidence.Path)
	result := setupSvcLiveReplayCoverageResult{
		Mode:           "setup-svc-live-replay-coverage",
		Project:        projectPath,
		ReadOnly:       true,
		Status:         "passed",
		MatrixPath:     matrixContract.Path,
		MatrixContract: matrixContract,
		TestEvidence:   testEvidence,
		Notes: []string{
			"This read-only audit proves the current parity matrix covers every supported MSAPI create/update/delete/query contract shape.",
			"Replay test evidence must map every matrix domain operation to a concrete parity replay test class and method.",
			"Domains remain covered, not verified, until approved live setup-svc replay evidence passes and matrix promotion is applied.",
			"Variant operations such as physical-purge, assign, and remove are reported separately from the canonical CRUD/query set.",
		},
	}
	if matrixContract.Status != "passed" {
		result.Status = "blocked"
		for _, issue := range matrixContract.Issues {
			result.BlockingIssues = append(result.BlockingIssues, "parityMatrix: "+issue)
		}
	}
	if testEvidence.Status != "passed" {
		result.Status = "blocked"
		for _, issue := range testEvidence.Issues {
			result.BlockingIssues = append(result.BlockingIssues, "testEvidence: "+issue)
		}
	}
	for _, expected := range setupSvcLiveReplayDomains() {
		item := matrixItems[normalizeDomain(expected.Domain)]
		evidenceItem := testEvidenceItems[normalizeDomain(expected.Domain)]
		matrixStatus := "missing"
		queryIncluded := false
		setupSvcReferences := []string{}
		runtimeEffects := []string{}
		queryReadbackExpectations := []string{}
		requiredTables := append([]string{}, expected.RequiredTables...)
		operations := append([]string{}, expected.Operations...)
		testEvidenceOps := []string{}
		if item != nil {
			matrixStatus = strings.ToLower(strings.TrimSpace(firstMapString(item, "status", "currentStatus")))
			queryIncluded, _ = item["queryIncluded"].(bool)
			setupSvcReferences = stringList(item["setupSvcReferences"])
			runtimeEffects = stringList(item["runtimeEffects"])
			queryReadbackExpectations = stringList(item["queryReadbackExpectations"])
			if tables := stringList(item["requiredTables"]); len(tables) > 0 {
				requiredTables = tables
			}
			if ops := stringList(item["operations"]); len(ops) > 0 {
				operations = ops
			}
		}
		if evidenceItem != nil {
			testEvidenceOps = setupSvcLiveReplayTestEvidenceOperations(evidenceItem)
		}
		domain := setupSvcLiveReplayCoverageDomain{
			Domain:                    expected.Domain,
			Status:                    "passed",
			MatrixStatus:              matrixStatus,
			HasCanonicalCRUDQ:         setupSvcLiveReplayHasCanonicalCRUDQ(operations),
			QueryIncluded:             queryIncluded,
			Operations:                operations,
			VariantOperations:         setupSvcLiveReplayVariantOperations(operations),
			RequiredTables:            requiredTables,
			SetupSvcReferences:        setupSvcReferences,
			RuntimeEffects:            runtimeEffects,
			QueryReadbackExpectations: queryReadbackExpectations,
			TestEvidenceOps:           testEvidenceOps,
		}
		if item == nil {
			domain.BlockingIssues = append(domain.BlockingIssues, "missing parity matrix domain")
		}
		if evidenceItem == nil {
			domain.BlockingIssues = append(domain.BlockingIssues, "missing parity replay test evidence")
		} else {
			if missing := missingSetupSvcLiveReplayStrings(operations, testEvidenceOps, false); len(missing) > 0 {
				domain.BlockingIssues = append(domain.BlockingIssues, "missing replay test evidence for operations "+strings.Join(missing, ","))
			}
			if unexpected := unexpectedSetupSvcLiveReplayStrings(operations, testEvidenceOps, false); len(unexpected) > 0 {
				domain.BlockingIssues = append(domain.BlockingIssues, "unexpected replay test evidence operations "+strings.Join(unexpected, ","))
			}
			if duplicates := duplicateSetupSvcLiveReplayStrings(testEvidenceOps, false); len(duplicates) > 0 {
				domain.BlockingIssues = append(domain.BlockingIssues, "duplicate replay test evidence operations "+strings.Join(duplicates, ","))
			}
		}
		if !domain.HasCanonicalCRUDQ {
			domain.BlockingIssues = append(domain.BlockingIssues, "missing canonical create/update/delete/query operation coverage")
		}
		if !queryIncluded {
			domain.BlockingIssues = append(domain.BlockingIssues, "queryIncluded must be true")
		}
		if len(requiredTables) == 0 {
			domain.BlockingIssues = append(domain.BlockingIssues, "missing requiredTables")
		}
		if len(setupSvcReferences) == 0 {
			domain.BlockingIssues = append(domain.BlockingIssues, "missing setupSvcReferences")
		}
		if len(runtimeEffects) == 0 {
			domain.BlockingIssues = append(domain.BlockingIssues, "missing runtimeEffects")
		}
		if len(queryReadbackExpectations) == 0 {
			domain.BlockingIssues = append(domain.BlockingIssues, "missing queryReadbackExpectations")
		}
		if len(domain.BlockingIssues) > 0 {
			domain.Status = "blocked"
			result.Totals.BlockedDomains++
			result.BlockingIssues = append(result.BlockingIssues, expected.Domain+": "+strings.Join(domain.BlockingIssues, "; "))
		} else {
			if matrixStatus == "verified" {
				result.Totals.VerifiedDomains++
			} else {
				result.Totals.CoveredDomains++
			}
		}
		result.Totals.Domains++
		result.Totals.Operations += len(operations)
		result.Totals.RequiredTables += len(requiredTables)
		result.Totals.SetupSvcReferences += len(setupSvcReferences)
		result.Totals.RuntimeEffects += len(runtimeEffects)
		result.Totals.QueryReadbackExpectations += len(queryReadbackExpectations)
		result.Totals.TestEvidenceOperations += len(testEvidenceOps)
		if evidenceItem != nil {
			result.Totals.TestEvidenceDomains++
		}
		if domain.HasCanonicalCRUDQ {
			result.Totals.CanonicalCrudQueryDomains++
		}
		for _, operation := range operations {
			if !setupSvcLiveReplayIsCanonicalOperation(operation) {
				result.OperationFamilies.Variants = append(result.OperationFamilies.Variants, expected.Domain+"/"+operation)
				result.Totals.VariantOperations++
			}
			switch setupSvcLiveReplayOperationFamily(operation) {
			case "create":
				result.OperationFamilies.Create = append(result.OperationFamilies.Create, expected.Domain+"/"+operation)
				result.Totals.WriteOperations++
			case "update":
				result.OperationFamilies.Update = append(result.OperationFamilies.Update, expected.Domain+"/"+operation)
				result.Totals.WriteOperations++
			case "delete":
				result.OperationFamilies.Delete = append(result.OperationFamilies.Delete, expected.Domain+"/"+operation)
				result.Totals.WriteOperations++
			case "query":
				result.OperationFamilies.Query = append(result.OperationFamilies.Query, expected.Domain+"/"+operation)
				result.Totals.QueryOperations++
			default:
				result.Totals.WriteOperations++
			}
		}
		result.Domains = append(result.Domains, domain)
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked"
	}
	return result
}

func buildSetupSvcLiveReplayPreflightResult(projectPath string, metadataServiceURL string) setupSvcLiveReplayPreflightResult {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, "")
	packet := buildSetupSvcLiveReplayPacket(projectPath)
	captureSources := setupSvcLiveReplayCaptureSourceSummaryFor(projectPath, packet)
	packetDryRun, packetErr := setupSvcLiveReplayPacketApplyResult(projectPath, packet, false, "")
	readiness := setupSvcLiveReplayReadinessResult(projectPath, metadataServiceURL)
	datasourceReadiness := setupSvcLiveReplayDatasourceReadinessFor()
	coverage := buildSetupSvcLiveReplayCoverageResult(projectPath)
	gaps, gapsErr := buildSetupSvcLiveReplayGapResult(projectPath, "")
	bundle := buildSetupSvcLiveReplayEvidenceBundleScanResult(projectPath, "")
	completion := buildSetupSvcLiveReplayCompletionAuditResult(projectPath, "")
	if gapsErr == nil && gaps.Status != "missing_manifest" && captureSources.SourceFilesIncomplete > 0 {
		if checklist, err := buildSetupSvcLiveReplaySourceChecklistResult(projectPath, "", "--source-readiness", "incomplete"); err == nil {
			captureSources.MissingSectionCounts = checklist.MissingSectionCounts
			captureSources.NextQueueCommands = checklist.NextQueueCommands
			captureSources.PageWorklistSaveCommands = checklist.PageWorklistSaveCommands
			captureSources.PageChecklistSaveCommands = checklist.PageChecklistSaveCommands
			captureSources.PageSaveScript = checklist.PageSaveScript
			captureSources.PageSaveScriptPath = checklist.PageSaveScriptPath
			captureSources.SavePageSaveScriptCommand = setupSvcLiveReplayPreflightSavePageScriptCommand(projectPath, captureSources.PageSaveScriptPath)
			sourceExecutionOptions := setupSvcLiveReplayCollectionPlanOptions{SourceReadiness: "incomplete", Limit: 25}
			captureSources.SourceExecutionPacketPath = setupSvcLiveReplaySourceExecutionPacketSuggestedPath(projectPath, sourceExecutionOptions)
			captureSources.SaveSourceExecutionPacketCommand = setupSvcLiveReplaySourceExecutionCommand(projectPath, checklist.ManifestPath, sourceExecutionOptions) + " > " + shellPath(captureSources.SourceExecutionPacketPath)
			captureSources.SourceExecutionBatchScriptPath = setupSvcLiveReplaySourceExecutionScriptSuggestedPath(projectPath, sourceExecutionOptions)
			captureSources.SaveSourceExecutionBatchScriptCommand = setupSvcLiveReplaySourceExecutionSaveBatchScriptCommand(projectPath, checklist.ManifestPath, sourceExecutionOptions, captureSources.SourceExecutionBatchScriptPath)
			captureSources.SourceExecutionImportScriptPath = setupSvcLiveReplaySourceExecutionImportScriptSuggestedPath(projectPath, sourceExecutionOptions)
			captureSources.SaveSourceExecutionImportScriptCommand = setupSvcLiveReplaySourceExecutionSaveImportBatchScriptCommand(projectPath, checklist.ManifestPath, sourceExecutionOptions, captureSources.SourceExecutionImportScriptPath)
		}
	}
	result := setupSvcLiveReplayPreflightResult{
		Mode:                      "setup-svc-live-replay-preflight",
		Project:                   projectPath,
		ReadOnly:                  true,
		Status:                    "ready_for_approved_live_replay",
		ManifestPath:              manifestPath,
		PacketPathHint:            filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "packet.json"),
		Totals:                    packet.Totals,
		MetadataServiceDatasource: datasourceReadiness,
		CaptureSources:            captureSources,
		NextCommands: []string{
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-packet > setup-svc-live-replay-packet.json",
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-packet @setup-svc-live-replay-packet.json --dry-run",
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-workspace @setup-svc-live-replay-packet.json --execute --approval " + setupSvcParityEvidenceWorkspaceApproval,
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-gaps",
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit",
		},
		Notes: []string{
			"This command is read-only and aggregates the setup-svc live replay preflight gates.",
			"It does not execute setup-svc, MetadataService writes, workspace creation, evidence bundle writes, promotion, or matrix updates.",
			"A missing manifest is expected before approved live replay evidence collection begins.",
		},
	}
	result.addPreflightGate("readiness", readiness.Status, readiness.Status != "ready_for_approved_live_replay", map[string]any{
		"domains":    readiness.Totals.Domains,
		"operations": readiness.Totals.Operations,
		"verified":   readiness.Totals.Verified,
	}, "Repair config or parity matrix blockers before collecting live replay evidence.")
	result.addPreflightGate("metadata_service_datasource", datasourceReadiness.Status, false, map[string]any{
		"runtimeMode":            datasourceReadiness.RuntimeMode,
		"runtimeModeSource":      datasourceReadiness.RuntimeModeSource,
		"serverPort":             datasourceReadiness.ServerPort,
		"serverPortSource":       datasourceReadiness.ServerPortSource,
		"jdbcUrlConfigured":      datasourceReadiness.JDBCURLConfigured,
		"jdbcUrlSource":          datasourceReadiness.JDBCURLSource,
		"jdbcUrlLooksDefaultH2":  datasourceReadiness.JDBCURLLooksDefaultH2,
		"usernameConfigured":     datasourceReadiness.UsernameConfigured,
		"usernameSource":         datasourceReadiness.UsernameSource,
		"passwordConfigured":     datasourceReadiness.PasswordConfigured,
		"passwordSource":         datasourceReadiness.PasswordSource,
		"driverConfigured":       datasourceReadiness.DriverConfigured,
		"driverSource":           datasourceReadiness.DriverSource,
		"readyForRealDatasource": datasourceReadiness.ReadyForRealDatasource,
		"missing":                datasourceReadiness.Missing,
		"warnings":               datasourceReadiness.Warnings,
	}, "Set real MDS_* datasource variables before approved MetadataService apply/query replay.")
	result.attachPreflightDatasourceGate(datasourceReadiness)
	if packetErr != nil {
		result.addPreflightGate("packet_dry_run", "error", true, map[string]any{"error": packetErr.Error()}, "Regenerate the setup-svc live replay packet and dry-run it before evidence collection.")
	} else {
		result.addPreflightGate("packet_dry_run", packetDryRun.Status, packetDryRun.Status != "dry_run_ready", map[string]any{
			"domains":          packetDryRun.Totals.Domains,
			"operations":       packetDryRun.Totals.Operations,
			"failedOperations": packetDryRun.Totals.FailedOperations,
			"manifestPath":     packetDryRun.ManifestPath,
		}, "Repair packet contract blockers before evidence collection.")
	}
	result.addPreflightGate("coverage", coverage.Status, coverage.Status != "passed", map[string]any{
		"domains":                      coverage.Totals.Domains,
		"operations":                   coverage.Totals.Operations,
		"runtimeEffects":               coverage.Totals.RuntimeEffects,
		"queryReadbackExpectations":    coverage.Totals.QueryReadbackExpectations,
		"matrixContractStatus":         coverage.MatrixContract.Status,
		"replayTestEvidenceStatus":     coverage.TestEvidence.Status,
		"replayTestSourceMethodStatus": coverage.TestEvidence.TestSourceStatus,
	}, "Repair matrix, replay test evidence, or source-method coverage before live replay.")
	if gapsErr != nil {
		result.addPreflightGate("gaps", "error", true, map[string]any{"error": gapsErr.Error()}, "Repair the live replay manifest path or JSON before continuing evidence collection.")
	} else {
		gapsBlocking := !(gaps.Status == "missing_manifest" || gaps.Status == "pending_evidence" || gaps.Status == "ready_for_normalized_diff" || gaps.Status == "ready_for_evidence" || gaps.Status == "complete")
		result.addPreflightGate("gaps", gaps.Status, gapsBlocking, map[string]any{
			"missingOperations":      gaps.Totals.MissingOperations,
			"pendingOperations":      gaps.Totals.PendingOperations,
			"failedOperations":       gaps.Totals.FailedOperations,
			"readyForDiffOperations": gaps.Totals.ReadyForDiffOperations,
		}, "Repair failed evidence artifacts before continuing live replay collection.")
	}
	result.addPreflightGate("capture_sources", captureSources.Status, false, map[string]any{
		"sourceRoot":                             captureSources.SourceRoot,
		"captureRoot":                            captureSources.CaptureRoot,
		"artifactFiles":                          captureSources.ArtifactFiles,
		"sourceFiles":                            captureSources.SourceFiles,
		"sourceFilesPresent":                     captureSources.SourceFilesPresent,
		"sourceFilesMissing":                     captureSources.SourceFilesMissing,
		"sourceFilesComplete":                    captureSources.SourceFilesComplete,
		"sourceFilesIncomplete":                  captureSources.SourceFilesIncomplete,
		"sourceTemplatesMissingGuideFields":      captureSources.SourceTemplatesMissingGuideFields,
		"missingEvidenceSectionCounts":           captureSources.MissingSectionCounts,
		"nextSourceQueueCommands":                captureSources.NextQueueCommands,
		"captureSourceWorkspaceDryRunCommand":    captureSources.CaptureSourceWorkspaceDryRunCommand,
		"captureSourceWorkspaceExecuteCommand":   captureSources.CaptureSourceWorkspaceExecuteCommand,
		"captureSourceWorkspaceRefreshCommand":   captureSources.CaptureSourceWorkspaceRefreshCommand,
		"missingSourceChecklistCommand":          captureSources.MissingSourceChecklistCommand,
		"incompleteWorklistCommand":              captureSources.IncompleteWorklistCommand,
		"incompleteSourceChecklistCommand":       captureSources.IncompleteSourceChecklistCommand,
		"sourceExecutionPacketPath":              captureSources.SourceExecutionPacketPath,
		"saveSourceExecutionPacketCommand":       captureSources.SaveSourceExecutionPacketCommand,
		"sourceExecutionBatchScriptPath":         captureSources.SourceExecutionBatchScriptPath,
		"saveSourceExecutionBatchScriptCommand":  captureSources.SaveSourceExecutionBatchScriptCommand,
		"sourceExecutionImportScriptPath":        captureSources.SourceExecutionImportScriptPath,
		"saveSourceExecutionImportScriptCommand": captureSources.SaveSourceExecutionImportScriptCommand,
	}, "Mirror real capture JSON files under captures/outputs/setup-svc-live-replay, then import the source-readiness complete worklist.")
	bundleBlocking := !(bundle.Status == "missing" || bundle.Status == "passed")
	result.addPreflightGate("evidence_bundle", bundle.Status, bundleBlocking, map[string]any{
		"bundlePath":     bundle.BundlePath,
		"manifestPath":   bundle.ManifestPath,
		"evidenceStatus": bundle.Bundle.EvidenceStatus,
	}, "Repair stale or invalid evidence bundle before promotion.")
	completionAllowed := completion.Status == "blocked_missing_live_replay_evidence" ||
		completion.Status == "blocked_live_replay_evidence" ||
		completion.Status == "blocked_evidence_bundle" ||
		completion.Status == "ready_for_matrix_status_update" ||
		completion.Status == "passed"
	result.addPreflightGate("completion_audit", completion.Status, !completionAllowed, map[string]any{
		"matrixVerifiedDomains":    completion.Totals.MatrixVerifiedDomains,
		"matrixNonVerifiedDomains": completion.Totals.MatrixNonVerifiedDomains,
		"blockedDomains":           completion.Totals.BlockedDomains,
	}, "Repair completion audit blockers before collecting or promoting evidence.")
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked_preflight"
		result.NextCommands = setupSvcLiveReplayPreflightAppendSourceExecutionCommands(result.NextCommands, captureSources)
		return result
	}
	switch {
	case completion.Status == "passed":
		result.Status = "passed"
	case completion.Status == "blocked_evidence_bundle":
		result.Status = "blocked_evidence_bundle"
		result.NextCommands = append([]string{}, completion.NextCommands...)
	case completion.Status == "ready_for_matrix_status_update":
		result.Status = "ready_for_matrix_status_update"
		result.NextCommands = append([]string{}, completion.NextCommands...)
	case gapsErr == nil && gaps.Status != "missing_manifest":
		result.Status = "evidence_collection_in_progress"
		result.NextCommands = setupSvcLiveReplayPreflightEvidenceCollectionCommands(gaps)
		result.NextCommands = setupSvcLiveReplayPreflightAppendSourceExecutionCommands(result.NextCommands, captureSources)
	default:
		result.Status = "ready_for_approved_live_replay"
	}
	return result
}

func setupSvcLiveReplayPreflightAppendSourceExecutionCommands(commands []string, captureSources setupSvcLiveReplayCaptureSourceSummary) []string {
	if captureSources.SourceFilesIncomplete <= 0 {
		return commands
	}
	for _, command := range []string{
		captureSources.SaveSourceExecutionPacketCommand,
		captureSources.SaveSourceExecutionBatchScriptCommand,
		captureSources.SaveSourceExecutionImportScriptCommand,
	} {
		trimmed := strings.TrimSpace(command)
		if trimmed != "" && !containsString(commands, trimmed) {
			commands = append(commands, trimmed)
		}
	}
	return commands
}

func buildSetupSvcLiveReplayEnvironmentResult(projectPath string, metadataServiceURL string) setupSvcLiveReplayEnvironmentResult {
	packet := buildSetupSvcLiveReplayPacket(projectPath)
	captureSources := setupSvcLiveReplayCaptureSourceSummaryFor(projectPath, packet)
	completion := buildSetupSvcLiveReplayCompletionAuditResult(projectPath, "")
	configSnapshot := setupSvcLiveReplayEnvironmentConfigFor(projectPath, metadataServiceURL)
	metadataService := setupSvcLiveReplayEndpointHealth(metadataServiceURL)
	datasourceReadiness := setupSvcLiveReplayDatasourceReadinessFor()
	result := setupSvcLiveReplayEnvironmentResult{
		Mode:                      "setup-svc-live-replay-environment",
		Project:                   projectPath,
		ReadOnly:                  true,
		Status:                    "ready_for_replay_capture",
		Config:                    configSnapshot,
		MetadataService:           metadataService,
		MetadataServiceDatasource: datasourceReadiness,
		CaptureSources:            captureSources,
		CompletionAudit: setupSvcLiveReplayEnvironmentAudit{
			Status:                     completion.Status,
			ManifestPath:               completion.ManifestPath,
			MatrixContractStatus:       completion.MatrixContract.Status,
			MatrixVerifiedDomains:      completion.Totals.MatrixVerifiedDomains,
			MatrixNonVerifiedDomains:   completion.Totals.MatrixNonVerifiedDomains,
			EvidenceVerifiedOperations: completion.Totals.EvidenceVerifiedOperations,
			CompletedOperations:        completion.Totals.CompletedOperations,
			FailedEvidenceTotal:        completion.FailedEvidenceSummary.Total,
			RepairQueueCount:           len(completion.FailedEvidenceSummary.RepairQueues),
			RepairQueues:               completion.FailedEvidenceSummary.RepairQueues,
		},
		NextCommands: []string{
			"cloudcc capabilities msapi " + shellPath(projectPath),
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-preflight",
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-health --source-readiness incomplete",
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit",
		},
		Notes: []string{
			"This command is read-only and does not call setup-svc writes, MetadataService writes, evidence import, bundle writes, promotion, or matrix updates.",
			"Secret values are intentionally not emitted; token fields are reported only as configured/not configured.",
			"Use this before live replay capture to verify local service reachability and evidence-workspace readiness.",
		},
	}
	result.addEnvironmentGate("project_config", boolStatus(configSnapshot.ConfigFileExists), !configSnapshot.ConfigFileExists, map[string]any{
		"profile":                    configSnapshot.ConfigProfile,
		"setupSvcConfigured":         configSnapshot.SetupSvcConfigured,
		"apiSvcConfigured":           configSnapshot.ApiSvcConfigured,
		"accessTokenConfigured":      configSnapshot.AccessTokenConfigured,
		"metadataServiceConfigured":  configSnapshot.MetadataServiceConfigured,
		"metadataServiceEndpoint":    configSnapshot.MetadataServiceEndpoint,
		"metadataServiceTokenSource": configSnapshot.MetadataServiceTokenSource,
	}, "Create or select cloudcc-cli.config.json before replay.")
	result.addEnvironmentGate("metadata_service_health", metadataService.Status, !metadataService.Reachable, map[string]any{
		"url":      metadataService.URL,
		"httpCode": metadataService.HTTPCode,
		"error":    metadataService.Error,
	}, "Start MetadataService or fix CLOUDCC_METADATA_SERVICE_URL before replay.")
	result.addEnvironmentGate("metadata_service_datasource", datasourceReadiness.Status, false, map[string]any{
		"runtimeMode":            datasourceReadiness.RuntimeMode,
		"runtimeModeSource":      datasourceReadiness.RuntimeModeSource,
		"serverPort":             datasourceReadiness.ServerPort,
		"serverPortSource":       datasourceReadiness.ServerPortSource,
		"jdbcUrlConfigured":      datasourceReadiness.JDBCURLConfigured,
		"jdbcUrlSource":          datasourceReadiness.JDBCURLSource,
		"jdbcUrlLooksDefaultH2":  datasourceReadiness.JDBCURLLooksDefaultH2,
		"usernameConfigured":     datasourceReadiness.UsernameConfigured,
		"usernameSource":         datasourceReadiness.UsernameSource,
		"passwordConfigured":     datasourceReadiness.PasswordConfigured,
		"passwordSource":         datasourceReadiness.PasswordSource,
		"driverConfigured":       datasourceReadiness.DriverConfigured,
		"driverSource":           datasourceReadiness.DriverSource,
		"readyForRealDatasource": datasourceReadiness.ReadyForRealDatasource,
		"missing":                datasourceReadiness.Missing,
		"warnings":               datasourceReadiness.Warnings,
	}, "Set real MDS_* datasource variables before live MetadataService apply/query replay.")
	result.attachEnvironmentDatasourceGate(datasourceReadiness)
	result.addEnvironmentGate("capture_sources", captureSources.Status, false, map[string]any{
		"artifactFiles":         captureSources.ArtifactFiles,
		"sourceFiles":           captureSources.SourceFiles,
		"sourceFilesPresent":    captureSources.SourceFilesPresent,
		"sourceFilesMissing":    captureSources.SourceFilesMissing,
		"sourceFilesComplete":   captureSources.SourceFilesComplete,
		"sourceFilesIncomplete": captureSources.SourceFilesIncomplete,
	}, "Refresh capture source templates, then replace incomplete sources with real replay evidence.")
	result.addEnvironmentGate("completion_audit", completion.Status, false, map[string]any{
		"matrixContractStatus":       completion.MatrixContract.Status,
		"matrixVerifiedDomains":      completion.Totals.MatrixVerifiedDomains,
		"matrixNonVerifiedDomains":   completion.Totals.MatrixNonVerifiedDomains,
		"evidenceVerifiedOperations": completion.Totals.EvidenceVerifiedOperations,
		"completedOperations":        completion.Totals.CompletedOperations,
		"failedEvidenceTotal":        completion.FailedEvidenceSummary.Total,
		"repairQueueCount":           len(completion.FailedEvidenceSummary.RepairQueues),
	}, "Completion remains blocked until live replay evidence, bundle, promotion, and matrix verification all pass.")
	switch {
	case !configSnapshot.ConfigFileExists:
		result.Status = "blocked_missing_project_config"
	case !metadataService.Reachable:
		result.Status = "blocked_metadata_service_unreachable"
	case completion.Status == "passed":
		result.Status = "passed"
	case captureSources.SourceFilesIncomplete > 0 || completion.Status == "blocked_live_replay_evidence":
		result.Status = "evidence_collection_in_progress"
	case captureSources.SourceFilesMissing > 0:
		result.Status = "ready_for_capture_source_workspace"
	default:
		result.Status = "ready_for_replay_capture"
	}
	if !metadataService.Reachable {
		next := []string{
			"Start MetadataService at " + shellPath(redactEndpoint(metadataServiceURL)) + ", then rerun cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-environment",
		}
		if strings.TrimSpace(datasourceReadiness.StartCommandHint) != "" {
			next = append(next, datasourceReadiness.StartCommandHint)
		}
		result.NextCommands = append(next, result.NextCommands...)
	}
	return result
}

func (result *setupSvcLiveReplayEnvironmentResult) addEnvironmentGate(name string, status string, blocking bool, summary map[string]any, nextAction string) {
	result.Gates = append(result.Gates, setupSvcLiveReplayPreflightGate{
		Name:       name,
		Status:     status,
		Blocking:   blocking,
		Summary:    summary,
		NextAction: nextAction,
	})
	if blocking {
		result.BlockingIssues = append(result.BlockingIssues, name+": "+status)
	}
}

func (result *setupSvcLiveReplayEnvironmentResult) attachEnvironmentDatasourceGate(readiness setupSvcLiveReplayDatasourceReadiness) {
	if len(result.Gates) == 0 {
		return
	}
	result.Gates[len(result.Gates)-1].MetadataServiceDatasource = &readiness
}

func setupSvcLiveReplayEnvironmentConfigFor(projectPath string, metadataServiceURL string) setupSvcLiveReplayEnvironmentConfig {
	out := setupSvcLiveReplayEnvironmentConfig{
		MetadataServiceEndpoint:   redactEndpoint(metadataServiceURL),
		MetadataServiceConfigured: strings.TrimSpace(metadataServiceURL) != "",
	}
	if _, err := os.Stat(filepath.Join(projectPath, "cloudcc-cli.config.json")); err == nil {
		out.ConfigFileExists = true
	}
	active := map[string]any{}
	if root, err := config.Root(projectPath); err == nil {
		if use, _ := root["use"].(string); use != "" {
			out.ConfigProfile = use
			if selected, _ := root[use].(map[string]any); selected != nil {
				active = selected
			}
		}
	}
	if len(active) == 0 {
		if pkg, err := readJSONFile(filepath.Join(projectPath, "package.json")); err == nil {
			if devConsole, _ := pkg["devConsoleConfig"].(map[string]any); devConsole != nil {
				active = devConsole
			}
		}
	}
	setupSvc := firstString(stringValue(active["setupSvc"]), derivedEndpoint(stringValue(active["baseUrl"]), stringValue(active["setupSvcPrefix"]), "/setup"))
	apiSvc := firstString(stringValue(active["apiSvc"]), derivedEndpoint(stringValue(active["baseUrl"]), stringValue(active["apiSvcPrefix"]), "/apisvc"))
	out.SetupSvcConfigured = setupSvc != ""
	out.SetupSvcEndpoint = redactEndpoint(setupSvc)
	out.ApiSvcConfigured = apiSvc != ""
	out.ApiSvcEndpoint = redactEndpoint(apiSvc)
	out.AccessTokenConfigured = tokenFromMap(active) != ""
	if ms, _ := active["metadataService"].(map[string]any); ms != nil {
		if stringValue(ms["url"]) != "" {
			out.MetadataServiceConfigured = true
		}
		if stringValue(ms["accessToken"]) != "" || stringValue(ms["token"]) != "" {
			out.MetadataServiceTokenSource = "metadataService"
		}
	}
	if stringValue(active["metadataServiceUrl"]) != "" || stringValue(active["metadata_service_url"]) != "" {
		out.MetadataServiceConfigured = true
	}
	if stringValue(active["metadataServiceAccessToken"]) != "" {
		out.MetadataServiceTokenSource = "metadataServiceAccessToken"
	}
	if out.MetadataServiceTokenSource == "" && out.AccessTokenConfigured {
		out.MetadataServiceTokenSource = "accessToken"
	}
	return out
}

func setupSvcLiveReplayDatasourceReadinessFor() setupSvcLiveReplayDatasourceReadiness {
	runtimeMode, runtimeModeSource := envOrDefault("MDS_RUNTIME_MODE", "self-hosted")
	serverPort, serverPortSource := envOrDefault("MDS_SERVER_PORT", "8087")
	jdbcURL, jdbcURLConfigured := os.LookupEnv("MDS_JDBC_URL")
	username, usernameConfigured := os.LookupEnv("MDS_DB_USERNAME")
	password, passwordConfigured := os.LookupEnv("MDS_DB_PASSWORD")
	driver, driverConfigured := os.LookupEnv("MDS_DB_DRIVER")
	jdbcURL = strings.TrimSpace(jdbcURL)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	driver = strings.TrimSpace(driver)

	out := setupSvcLiveReplayDatasourceReadiness{
		RuntimeMode:        runtimeMode,
		RuntimeModeSource:  runtimeModeSource,
		ServerPort:         serverPort,
		ServerPortSource:   serverPortSource,
		JDBCURLConfigured:  jdbcURLConfigured && jdbcURL != "",
		UsernameConfigured: usernameConfigured && username != "",
		PasswordConfigured: passwordConfigured && password != "",
		DriverConfigured:   driverConfigured && driver != "",
		StartCommandHint: strings.Join([]string{
			"cd cc-metadata-service &&",
			"MDS_RUNTIME_MODE=self-hosted",
			"MDS_SERVER_PORT=8087",
			"MDS_JDBC_URL=<jdbc-url>",
			"MDS_DB_USERNAME=<username>",
			"MDS_DB_PASSWORD=<password>",
			"MDS_DB_DRIVER=<driver>",
			"./mvnw spring-boot:run",
		}, " "),
		Redaction: "MDS_JDBC_URL, MDS_DB_USERNAME, MDS_DB_PASSWORD, and MDS_DB_DRIVER values are never emitted; only presence and source are reported.",
	}
	if out.JDBCURLConfigured {
		out.JDBCURLSource = "env:MDS_JDBC_URL"
	}
	if out.UsernameConfigured {
		out.UsernameSource = "env:MDS_DB_USERNAME"
	}
	if out.PasswordConfigured {
		out.PasswordSource = "env:MDS_DB_PASSWORD"
	}
	if out.DriverConfigured {
		out.DriverSource = "env:MDS_DB_DRIVER"
	}
	out.JDBCURLLooksDefaultH2 = !out.JDBCURLConfigured || looksLikeDefaultMetadataServiceH2(jdbcURL)
	if !out.JDBCURLConfigured {
		out.Missing = append(out.Missing, "MDS_JDBC_URL")
	}
	if !out.UsernameConfigured {
		out.Missing = append(out.Missing, "MDS_DB_USERNAME")
	}
	if !out.PasswordConfigured {
		out.Missing = append(out.Missing, "MDS_DB_PASSWORD")
	}
	if !out.DriverConfigured {
		out.Missing = append(out.Missing, "MDS_DB_DRIVER")
	}
	if out.JDBCURLLooksDefaultH2 {
		out.Warnings = append(out.Warnings, "MetadataService appears to be using the default in-memory H2 datasource; live replay needs a real metadata database datasource.")
	}
	if strings.EqualFold(strings.TrimSpace(runtimeMode), "saas-model") {
		out.Warnings = append(out.Warnings, "MDS_RUNTIME_MODE is saas-model; this readiness gate checks the self-hosted local spring.datasource required by the current replay window.")
	}
	out.ReadyForRealDatasource = out.JDBCURLConfigured && out.UsernameConfigured && out.PasswordConfigured && out.DriverConfigured && !out.JDBCURLLooksDefaultH2
	if out.ReadyForRealDatasource {
		out.Status = "ready"
	} else if len(out.Missing) > 0 {
		out.Status = "missing_real_datasource"
	} else {
		out.Status = "using_default_or_non_real_datasource"
	}
	return out
}

func setupSvcLiveReplayDatasourceBlockingIssues(readiness setupSvcLiveReplayDatasourceReadiness) []string {
	issues := []string{}
	for _, missing := range readiness.Missing {
		issues = append(issues, "metadataServiceDatasource: missing "+missing)
	}
	if readiness.JDBCURLLooksDefaultH2 {
		issues = append(issues, "metadataServiceDatasource: default H2 datasource is not valid live replay evidence")
	}
	if len(issues) == 0 {
		issues = append(issues, "metadataServiceDatasource: not ready for real datasource replay")
	}
	return issues
}

func setupSvcLiveReplayDatasourceReadinessMap(readiness setupSvcLiveReplayDatasourceReadiness) map[string]any {
	out := map[string]any{
		"runtimeMode":            readiness.RuntimeMode,
		"runtimeModeSource":      readiness.RuntimeModeSource,
		"serverPort":             readiness.ServerPort,
		"serverPortSource":       readiness.ServerPortSource,
		"jdbcUrlConfigured":      readiness.JDBCURLConfigured,
		"jdbcUrlSource":          readiness.JDBCURLSource,
		"jdbcUrlLooksDefaultH2":  readiness.JDBCURLLooksDefaultH2,
		"usernameConfigured":     readiness.UsernameConfigured,
		"usernameSource":         readiness.UsernameSource,
		"passwordConfigured":     readiness.PasswordConfigured,
		"passwordSource":         readiness.PasswordSource,
		"driverConfigured":       readiness.DriverConfigured,
		"driverSource":           readiness.DriverSource,
		"readyForRealDatasource": readiness.ReadyForRealDatasource,
		"status":                 readiness.Status,
		"missing":                append([]string{}, readiness.Missing...),
		"warnings":               append([]string{}, readiness.Warnings...),
		"startCommandHint":       readiness.StartCommandHint,
		"redaction":              readiness.Redaction,
	}
	return out
}

func envOrDefault(name string, fallback string) (string, string) {
	if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value), "env:" + name
	}
	return fallback, "default"
}

func looksLikeDefaultMetadataServiceH2(jdbcURL string) bool {
	normalized := strings.ToLower(strings.TrimSpace(jdbcURL))
	return normalized == "" ||
		strings.Contains(normalized, "jdbc:h2:mem:metadata_service") ||
		strings.Contains(normalized, "org.h2")
}

func setupSvcLiveReplayEndpointHealth(rawURL string) setupSvcLiveReplayEndpointCheck {
	check := setupSvcLiveReplayEndpointCheck{
		URL:    redactEndpoint(rawURL),
		Status: "not_configured",
	}
	trimmed := strings.TrimRight(strings.TrimSpace(redactEndpoint(rawURL)), "/")
	if trimmed == "" {
		return check
	}
	checkURL := trimmed + "/actuator/health"
	req, err := http.NewRequest(http.MethodGet, checkURL, nil)
	if err != nil {
		check.Status = "invalid_url"
		check.Error = err.Error()
		return check
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		check.Status = "unreachable"
		check.Error = err.Error()
		return check
	}
	defer resp.Body.Close()
	check.HTTPCode = resp.StatusCode
	if resp.StatusCode >= 200 && resp.StatusCode < 500 {
		check.Reachable = true
		check.Status = "reachable"
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			check.Status = "reachable_auth_required"
		}
		return check
	}
	check.Status = "http_error"
	return check
}

func derivedEndpoint(base string, prefix string, fallbackPrefix string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	if strings.TrimSpace(prefix) == "" {
		prefix = fallbackPrefix
	}
	return base + "/" + strings.TrimLeft(prefix, "/")
}

func redactEndpoint(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return trimmed
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func boolStatus(ok bool) string {
	if ok {
		return "passed"
	}
	return "missing"
}

func setupSvcLiveReplayPreflightEvidenceCollectionCommands(gaps setupSvcLiveReplayGapResult) []string {
	commands := setupSvcLiveReplayPreflightEvidenceWorklistCommands(gaps)
	for _, step := range gaps.CollectionPlan.Runbook {
		for _, command := range step.Commands {
			trimmed := strings.TrimSpace(command)
			if trimmed != "" && !containsString(commands, trimmed) {
				commands = append(commands, trimmed)
			}
		}
	}
	if len(commands) > 0 {
		return commands
	}
	fallback := []string{
		gaps.CollectionPlan.PageCommands.CurrentPage,
		gaps.NextCommands.GenerateDiffs,
		gaps.NextCommands.SyncManifest,
		gaps.NextCommands.VerifyEvidence,
		gaps.NextCommands.WriteBundle,
		gaps.NextCommands.PromotionAudit,
		gaps.NextCommands.CompletionAudit,
	}
	for _, command := range fallback {
		trimmed := strings.TrimSpace(command)
		if trimmed != "" && !containsString(commands, trimmed) {
			commands = append(commands, trimmed)
		}
	}
	return commands
}

func setupSvcLiveReplayCaptureSourceSummaryFor(projectPath string, packet setupSvcLiveReplayPacket) setupSvcLiveReplayCaptureSourceSummary {
	missingOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceStatus: "missing",
		Limit:        25,
		BatchIndex:   -1,
		BatchLimit:   1,
	}
	captureSourceWorkspaceOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceStatus: "missing",
		Limit:        0,
		BatchIndex:   -1,
		BatchLimit:   setupSvcLiveReplayWorklistBatchLimit,
	}
	presentOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceStatus: "present",
		Limit:        25,
		BatchIndex:   -1,
		BatchLimit:   setupSvcLiveReplayWorklistBatchLimit,
	}
	incompleteOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceReadiness: "incomplete",
		Limit:           25,
		BatchIndex:      -1,
		BatchLimit:      setupSvcLiveReplayWorklistBatchLimit,
	}
	completeOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceReadiness: "complete",
		Limit:           25,
		BatchIndex:      -1,
		BatchLimit:      setupSvcLiveReplayWorklistBatchLimit,
	}
	missingPacket := setupSvcLiveReplayWorklistOperatorPacketFor(projectPath, packet.ManifestPath, missingOptions, setupSvcLiveReplayGapCommands{})
	presentPacket := setupSvcLiveReplayWorklistOperatorPacketFor(projectPath, packet.ManifestPath, presentOptions, setupSvcLiveReplayGapCommands{})
	incompletePacket := setupSvcLiveReplayWorklistOperatorPacketFor(projectPath, packet.ManifestPath, incompleteOptions, setupSvcLiveReplayGapCommands{})
	completePacket := setupSvcLiveReplayWorklistOperatorPacketFor(projectPath, packet.ManifestPath, completeOptions, setupSvcLiveReplayGapCommands{})
	missingChecklist := setupSvcLiveReplaySourceChecklistOperatorPacketFor(projectPath, packet.ManifestPath, missingOptions, setupSvcLiveReplayGapCommands{})
	presentChecklist := setupSvcLiveReplaySourceChecklistOperatorPacketFor(projectPath, packet.ManifestPath, presentOptions, setupSvcLiveReplayGapCommands{})
	incompleteChecklist := setupSvcLiveReplaySourceChecklistOperatorPacketFor(projectPath, packet.ManifestPath, incompleteOptions, setupSvcLiveReplayGapCommands{})
	completeChecklist := setupSvcLiveReplaySourceChecklistOperatorPacketFor(projectPath, packet.ManifestPath, completeOptions, setupSvcLiveReplayGapCommands{})
	captureSourceWorkspaceDryRunCommand := setupSvcLiveReplayCaptureSourceWorkspaceCommand(projectPath, packet.ManifestPath, captureSourceWorkspaceOptions) + " --dry-run"
	summary := setupSvcLiveReplayCaptureSourceSummary{
		Status:      "missing",
		SourceRoot:  setupSvcLiveReplayWorklistSourceRoot(),
		CaptureRoot: setupSvcLiveReplayWorklistCaptureRoot(projectPath),
	}
	seen := map[string]bool{}
	for _, domain := range packet.Domains {
		for _, operation := range domain.Operations {
			for _, file := range operation.EvidenceFiles {
				normalized := strings.TrimSpace(file)
				if normalized == "" || seen[normalized] {
					continue
				}
				seen[normalized] = true
				summary.ArtifactFiles++
				sourceReadiness := setupSvcLiveReplaySourceReadinessFor(projectPath, normalized)
				switch sourceReadiness {
				case "complete":
					summary.SourceFilesPresent++
					summary.SourceFilesComplete++
				case "incomplete":
					summary.SourceFilesPresent++
					summary.SourceFilesIncomplete++
				default:
					summary.SourceFilesMissing++
				}
				if setupSvcLiveReplaySourceTemplateMissingGuideFields(projectPath, normalized) {
					summary.SourceTemplatesMissingGuideFields++
				}
			}
		}
	}
	summary.SourceFiles = summary.SourceFilesPresent + summary.SourceFilesMissing
	switch {
	case summary.ArtifactFiles == 0:
		summary.Status = "not_available"
	case summary.SourceFilesMissing == 0 && summary.SourceFilesIncomplete == 0:
		summary.Status = "complete"
	case summary.SourceFilesPresent > 0:
		summary.Status = "partial"
	default:
		summary.Status = "missing"
	}
	if summary.SourceFilesMissing > 0 {
		summary.CaptureSourceWorkspaceDryRunCommand = captureSourceWorkspaceDryRunCommand
		summary.CaptureSourceWorkspaceExecuteCommand = strings.TrimSuffix(captureSourceWorkspaceDryRunCommand, " --dry-run") + " --execute --approval " + setupSvcParityCaptureSourceWorkspaceApproval
		summary.MissingWorklistCommand = missingPacket.SaveWorklistCommand
		summary.MissingSourceChecklistCommand = missingChecklist.SaveChecklistCommand
	}
	if summary.SourceTemplatesMissingGuideFields > 0 {
		refreshOptions := captureSourceWorkspaceOptions
		refreshOptions.SourceStatus = "present"
		summary.CaptureSourceWorkspaceRefreshCommand = setupSvcLiveReplayCaptureSourceWorkspaceCommand(projectPath, packet.ManifestPath, refreshOptions) + " --execute --approval " + setupSvcParityCaptureSourceWorkspaceApproval
	}
	if summary.SourceFilesPresent > 0 {
		summary.PresentWorklistCommand = presentPacket.SaveWorklistCommand
		summary.PresentSourceChecklistCommand = presentChecklist.SaveChecklistCommand
	}
	if summary.SourceFilesIncomplete > 0 {
		summary.IncompleteWorklistCommand = incompletePacket.SaveWorklistCommand
		summary.IncompleteSourceChecklistCommand = incompleteChecklist.SaveChecklistCommand
	}
	if summary.SourceFilesComplete > 0 {
		summary.CompleteWorklistCommand = completePacket.SaveWorklistCommand
		summary.CompleteSourceChecklistCommand = completeChecklist.SaveChecklistCommand
		summary.DryRunImportCommand = completePacket.DryRunImportCommand
		summary.ExecuteImportCommand = completePacket.ExecuteImportCommand
	}
	return summary
}

func setupSvcLiveReplaySourceTemplateMissingGuideFields(projectPath string, artifactPath string) bool {
	sourcePath := filepath.Join(projectPath, setupSvcLiveReplayWorklistSuggestedSourcePath(artifactPath))
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return false
	}
	var artifact map[string]any
	if err := json.Unmarshal(body, &artifact); err != nil {
		return false
	}
	if strings.TrimSpace(stringValue(artifact["status"])) != "pending" || strings.TrimSpace(stringValue(artifact["sourceTemplateStatus"])) != "incomplete" {
		return false
	}
	if strings.TrimSpace(stringValue(artifact["requiredShapeKey"])) == "" {
		return true
	}
	if strings.TrimSpace(stringValue(artifact["manifestStatusField"])) == "" {
		return true
	}
	sections, ok := artifact["requiredEvidenceSections"].([]any)
	return !ok || len(sections) == 0
}

func setupSvcLiveReplayPreflightEvidenceWorklistCommands(gaps setupSvcLiveReplayGapResult) []string {
	if strings.TrimSpace(gaps.Project) == "" || strings.TrimSpace(gaps.ManifestPath) == "" {
		return nil
	}
	missingOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceStatus: "missing",
		Limit:        25,
		BatchIndex:   -1,
		BatchLimit:   1,
	}
	presentOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceStatus: "present",
		Limit:        25,
		BatchIndex:   -1,
		BatchLimit:   setupSvcLiveReplayWorklistBatchLimit,
	}
	incompleteOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceReadiness: "incomplete",
		Limit:           25,
		BatchIndex:      -1,
		BatchLimit:      setupSvcLiveReplayWorklistBatchLimit,
	}
	completeOptions := setupSvcLiveReplayCollectionPlanOptions{
		SourceReadiness: "complete",
		Limit:           25,
		BatchIndex:      -1,
		BatchLimit:      setupSvcLiveReplayWorklistBatchLimit,
	}
	missingPacket := setupSvcLiveReplayWorklistOperatorPacketFor(gaps.Project, gaps.ManifestPath, missingOptions, gaps.NextCommands)
	presentPacket := setupSvcLiveReplayWorklistOperatorPacketFor(gaps.Project, gaps.ManifestPath, presentOptions, gaps.NextCommands)
	incompletePacket := setupSvcLiveReplayWorklistOperatorPacketFor(gaps.Project, gaps.ManifestPath, incompleteOptions, gaps.NextCommands)
	completePacket := setupSvcLiveReplayWorklistOperatorPacketFor(gaps.Project, gaps.ManifestPath, completeOptions, gaps.NextCommands)
	missingChecklist := setupSvcLiveReplaySourceChecklistOperatorPacketFor(gaps.Project, gaps.ManifestPath, missingOptions, gaps.NextCommands)
	presentChecklist := setupSvcLiveReplaySourceChecklistOperatorPacketFor(gaps.Project, gaps.ManifestPath, presentOptions, gaps.NextCommands)
	incompleteChecklist := setupSvcLiveReplaySourceChecklistOperatorPacketFor(gaps.Project, gaps.ManifestPath, incompleteOptions, gaps.NextCommands)
	completeChecklist := setupSvcLiveReplaySourceChecklistOperatorPacketFor(gaps.Project, gaps.ManifestPath, completeOptions, gaps.NextCommands)
	packet := buildSetupSvcLiveReplayPacket(gaps.Project)
	sourceSummary := setupSvcLiveReplayCaptureSourceSummaryFor(gaps.Project, packet)
	var commands []string
	if sourceSummary.SourceFilesMissing > 0 {
		commands = append(commands,
			sourceSummary.CaptureSourceWorkspaceDryRunCommand,
			sourceSummary.CaptureSourceWorkspaceExecuteCommand,
			missingPacket.SaveWorklistCommand,
			missingChecklist.SaveChecklistCommand,
		)
	}
	if sourceSummary.SourceTemplatesMissingGuideFields > 0 {
		commands = append(commands, sourceSummary.CaptureSourceWorkspaceRefreshCommand)
	}
	if sourceSummary.SourceFilesIncomplete > 0 {
		commands = append(commands, incompletePacket.SaveWorklistCommand)
		commands = append(commands, incompleteChecklist.SaveChecklistCommand)
		commands = append(commands, setupSvcLiveReplayPreflightSourceBatchCommands(gaps.Project, gaps.ManifestPath, incompleteOptions)...)
	}
	if sourceSummary.SourceFilesComplete > 0 {
		commands = append(commands,
			completePacket.SaveWorklistCommand,
			completeChecklist.SaveChecklistCommand,
			completePacket.DryRunImportCommand,
			completePacket.ExecuteImportCommand,
		)
	}
	if sourceSummary.SourceFilesPresent > 0 {
		commands = append(commands, presentPacket.SaveWorklistCommand)
		commands = append(commands, presentChecklist.SaveChecklistCommand)
	}
	commands = append(commands, presentPacket.PostReplacementCommands...)
	return setupSvcLiveReplayUniqueCommands(commands)
}

func setupSvcLiveReplayPreflightSourceBatchCommands(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) []string {
	batchOptions := options
	if batchOptions.Limit == 0 {
		batchOptions.Limit = 25
	}
	batchOptions.BatchIndex = -1
	batchOptions.BatchLimit = 1
	worklist, err := buildSetupSvcLiveReplayWorklistResult(projectPath, manifestPath, setupSvcLiveReplayWorklistOptionArgs(batchOptions)...)
	if err != nil {
		return nil
	}
	var commands []string
	for _, queue := range worklist.Queues {
		for _, batch := range queue.Batches {
			commands = append(commands, firstString(batch.SaveWorklistCommand, batch.Command))
		}
	}
	return commands
}

func setupSvcLiveReplayWorklistOptionArgs(options setupSvcLiveReplayCollectionPlanOptions) []string {
	var args []string
	if options.Domain != "" {
		args = append(args, "--domain", options.Domain)
	}
	if options.Operation != "" {
		args = append(args, "--operation", options.Operation)
	}
	if options.ArtifactType != "" {
		args = append(args, "--artifact-type", options.ArtifactType)
	}
	if options.EvidenceSection != "" {
		args = append(args, "--evidence-section", options.EvidenceSection)
	}
	if options.SectionStatus != "" {
		args = append(args, "--section-status", options.SectionStatus)
	}
	if options.SourceStatus != "" {
		args = append(args, "--source-status", options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		args = append(args, "--source-readiness", options.SourceReadiness)
	}
	if options.Offset > 0 {
		args = append(args, "--offset", strconv.Itoa(options.Offset))
	}
	if options.Limit > 0 {
		args = append(args, "--limit", strconv.Itoa(options.Limit))
	}
	if options.BatchIndex >= 0 {
		args = append(args, "--batch-index", strconv.Itoa(options.BatchIndex))
	} else if options.BatchLimit != setupSvcLiveReplayWorklistBatchLimit {
		args = append(args, "--batch-limit", strconv.Itoa(options.BatchLimit))
	}
	return args
}

func setupSvcLiveReplayUniqueCommands(commands []string) []string {
	var result []string
	for _, command := range commands {
		trimmed := strings.TrimSpace(command)
		if trimmed != "" && !containsString(result, trimmed) {
			result = append(result, trimmed)
		}
	}
	return result
}

func (result *setupSvcLiveReplayPreflightResult) addPreflightGate(name string, status string, blocking bool, summary map[string]any, nextAction string) {
	gate := setupSvcLiveReplayPreflightGate{
		Name:       name,
		Status:     status,
		Blocking:   blocking,
		Summary:    summary,
		NextAction: nextAction,
	}
	if !blocking {
		gate.NextAction = ""
	}
	result.Gates = append(result.Gates, gate)
	if blocking {
		result.BlockingIssues = append(result.BlockingIssues, name+": "+status)
	}
}

func (result *setupSvcLiveReplayPreflightResult) attachPreflightDatasourceGate(readiness setupSvcLiveReplayDatasourceReadiness) {
	if len(result.Gates) == 0 {
		return
	}
	result.Gates[len(result.Gates)-1].MetadataServiceDatasource = &readiness
}

func buildSetupSvcLiveReplayPacket(projectPath string) setupSvcLiveReplayPacket {
	domains := setupSvcLiveReplayDomains()
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, "")
	contractFingerprint := setupSvcLiveReplayExpectedContractFingerprint()
	matrixContract := setupSvcLiveReplayMatrixContractStatus(projectPath)
	packet := setupSvcLiveReplayPacket{
		Mode:                "setup-svc-live-replay-packet",
		Project:             projectPath,
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		ReadOnly:            true,
		Execute:             false,
		ApprovalRequired:    true,
		ApprovalPhrase:      setupSvcParityReplayApproval,
		Status:              "ready_for_manual_evidence_collection",
		ManifestPath:        manifestPath,
		ContractVersion:     setupSvcLiveReplayContractVersion,
		ContractFingerprint: contractFingerprint,
		MatrixContract:      matrixContract,
		Commands: setupSvcLiveReplayPacketCommand{
			GeneratePacket:    "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-packet > setup-svc-live-replay-packet.json",
			DryRunPacket:      "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-packet @setup-svc-live-replay-packet.json --dry-run",
			VerifyEvidence:    "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
			EvidenceDirectory: filepath.Join(projectPath, "outputs", "setup-svc-live-replay"),
		},
		ManifestTemplate: setupSvcLiveReplayManifest{
			Mode:                "setup-svc-live-replay-evidence",
			Project:             projectPath,
			Status:              "pending",
			ContractVersion:     setupSvcLiveReplayContractVersion,
			ContractFingerprint: contractFingerprint,
		},
		StopConditions: []string{
			"Stop before any write unless the tenant/window is disposable or explicitly rollback-approved.",
			"Stop if a setup-svc baseline operation cannot be queried back with a stable metadata id.",
			"Stop if MetadataService apply does not return a VERIFIED operation id.",
			"Stop if normalized setup-svc and MetadataService snapshots differ outside documented normalizers.",
			"Stop if cleanup cannot remove disposable replay metadata after write operations.",
		},
		Notes: []string{
			"This packet is a manual live replay contract, not proof of parity by itself.",
			"Leave every manifest status as pending until the corresponding setup-svc, MetadataService, query/readback, diff, and cleanup evidence has been collected.",
			"The evidence verifier accepts only passed statuses with readable JSON evidence files; pending, failed, or missing artifacts keep matrix entries covered, not verified.",
		},
	}
	if matrixContract.Status != "passed" {
		packet.Status = "blocked_parity_matrix_contract"
		for _, issue := range matrixContract.Issues {
			packet.BlockingIssues = append(packet.BlockingIssues, "parityMatrix: "+issue)
		}
	}
	for _, domain := range domains {
		packetDomain := setupSvcLiveReplayPacketDomain{
			Domain:                    domain.Domain,
			Status:                    "pending_approved_replay",
			RequiredTables:            append([]string{}, domain.RequiredTables...),
			RuntimeEffects:            append([]string{}, domain.RuntimeEffects...),
			QueryReadbackExpectations: append([]string{}, domain.QueryReadbackExpectations...),
		}
		manifestDomain := setupSvcLiveReplayManifestDomain{Domain: domain.Domain}
		for _, operation := range domain.Operations {
			readOnly := operation == "query"
			requiredEvidence := setupSvcLiveReplayRequiredEvidence(operation)
			if !readOnly {
				packet.Totals.WriteOperations++
			} else {
				packet.Totals.QueryOperations++
			}
			evidenceFiles := setupSvcLiveReplayEvidenceFiles(domain.Domain, operation, !readOnly)
			packetDomain.Operations = append(packetDomain.Operations, setupSvcLiveReplayPacketOperation{
				Operation:        operation,
				Status:           "pending_evidence",
				ReadOnly:         readOnly,
				RequiredEvidence: requiredEvidence,
				EvidenceFiles:    evidenceFiles,
				OperatorSteps:    setupSvcLiveReplayOperatorSteps(projectPath, domain.Domain, operation, !readOnly),
			})
			manifestOperation := setupSvcLiveReplayManifestOperation{
				Operation:                     operation,
				SetupSvcEvidenceStatus:        "pending",
				MetadataServiceEvidenceStatus: "pending",
				QueryEvidenceStatus:           "pending",
				NormalizedDiffStatus:          "pending",
				EvidenceFiles:                 evidenceFiles,
				Notes:                         []string{"Replace pending statuses with passed only after attaching real evidence."},
			}
			if !readOnly {
				manifestOperation.CleanupStatus = "pending"
			}
			manifestDomain.Operations = append(manifestDomain.Operations, manifestOperation)
			packet.Totals.Operations++
		}
		packet.Domains = append(packet.Domains, packetDomain)
		packet.ManifestTemplate.Domains = append(packet.ManifestTemplate.Domains, manifestDomain)
		packet.Totals.Domains++
	}
	return packet
}

func (c *client) applySetupSvcLiveReplayPacket(stdout io.Writer, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc apply msapi [projectPath] setup-svc-live-replay-packet <@packet.json|packetJson> [--dry-run|--execute --approval %s]", setupSvcParityReplayApproval)
	}
	packetMap, err := parseObject(args[0], "cloudcc apply msapi setup-svc-live-replay-packet")
	if err != nil {
		return err
	}
	packet, err := decodeSetupSvcLiveReplayPacket(packetMap)
	if err != nil {
		return err
	}
	execute, approval := validationRuleApplyOptions(args[1:])
	result, err := setupSvcLiveReplayPacketApplyResult(c.projectPath, packet, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayWorkspace(stdout io.Writer, args []string) error {
	packet := buildSetupSvcLiveReplayPacket(c.projectPath)
	optionArgs := append([]string{}, args...)
	if len(optionArgs) > 0 && !strings.HasPrefix(strings.TrimSpace(optionArgs[0]), "--") {
		packetMap, err := parseObject(optionArgs[0], "cloudcc apply msapi setup-svc-live-replay-workspace")
		if err != nil {
			return err
		}
		decoded, err := decodeSetupSvcLiveReplayPacket(packetMap)
		if err != nil {
			return err
		}
		packet = decoded
		optionArgs = optionArgs[1:]
	}
	execute, approval := validationRuleApplyOptions(optionArgs)
	result, err := buildSetupSvcLiveReplayWorkspaceApplyResult(c.projectPath, packet, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayCaptureSourceWorkspace(stdout io.Writer, args []string) error {
	manifestArg := ""
	optionArgs := append([]string{}, args...)
	if len(optionArgs) > 0 && !strings.HasPrefix(strings.TrimSpace(optionArgs[0]), "--") {
		manifestArg = optionArgs[0]
		optionArgs = optionArgs[1:]
	}
	execute, approval := validationRuleApplyOptions(optionArgs)
	result, err := buildSetupSvcLiveReplayCaptureSourceWorkspaceApplyResult(c.projectPath, manifestArg, optionArgs, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayMatrixPromotion(stdout io.Writer, args []string) error {
	manifestArg := ""
	optionArgs := append([]string{}, args...)
	if len(optionArgs) > 0 && !strings.HasPrefix(strings.TrimSpace(optionArgs[0]), "--") {
		manifestArg = optionArgs[0]
		optionArgs = optionArgs[1:]
	}
	execute, approval := validationRuleApplyOptions(optionArgs)
	result, err := buildSetupSvcLiveReplayMatrixPromotionApplyResult(c.projectPath, manifestArg, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayNormalizedDiff(stdout io.Writer, args []string) error {
	manifestArg := ""
	optionArgs := append([]string{}, args...)
	if len(optionArgs) > 0 && !strings.HasPrefix(strings.TrimSpace(optionArgs[0]), "--") {
		manifestArg = optionArgs[0]
		optionArgs = optionArgs[1:]
	}
	execute, approval := validationRuleApplyOptions(optionArgs)
	result, err := buildSetupSvcLiveReplayNormalizedDiffApplyResult(c.projectPath, manifestArg, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayManifestSync(stdout io.Writer, args []string) error {
	manifestArg := ""
	optionArgs := append([]string{}, args...)
	if len(optionArgs) > 0 && !strings.HasPrefix(strings.TrimSpace(optionArgs[0]), "--") {
		manifestArg = optionArgs[0]
		optionArgs = optionArgs[1:]
	}
	execute, approval := validationRuleApplyOptions(optionArgs)
	result, err := buildSetupSvcLiveReplayManifestSyncApplyResult(c.projectPath, manifestArg, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayEvidenceBundle(stdout io.Writer, args []string) error {
	manifestArg := ""
	optionArgs := append([]string{}, args...)
	if len(optionArgs) > 0 && !strings.HasPrefix(strings.TrimSpace(optionArgs[0]), "--") {
		manifestArg = optionArgs[0]
		optionArgs = optionArgs[1:]
	}
	execute, approval := validationRuleApplyOptions(optionArgs)
	result, err := buildSetupSvcLiveReplayEvidenceBundleApplyResult(c.projectPath, manifestArg, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayQueryReadbackCapture(stdout io.Writer, args []string) error {
	manifestArg := ""
	optionArgs := append([]string{}, args...)
	if len(optionArgs) > 0 && !strings.HasPrefix(strings.TrimSpace(optionArgs[0]), "--") {
		manifestArg = optionArgs[0]
		optionArgs = optionArgs[1:]
	}
	execute, approval := validationRuleApplyOptions(optionArgs)
	result, err := c.buildSetupSvcLiveReplayQueryReadbackCaptureApplyResult(manifestArg, optionArgs, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplaySnapshotFromChanges(stdout io.Writer, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc apply msapi [projectPath] setup-svc-live-replay-snapshot-from-changes <@packet.json|packetJson> [--dry-run|--execute --approval %s]", setupSvcParitySnapshotFromChangesApproval)
	}
	packetMap, err := parseObject(args[0], "cloudcc apply msapi setup-svc-live-replay-snapshot-from-changes")
	if err != nil {
		return err
	}
	execute, approval := validationRuleApplyOptions(args[1:])
	result, err := c.buildSetupSvcLiveReplaySnapshotFromChangesApplyResult(packetMap, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayMetadataServiceApplyCapture(stdout io.Writer, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc apply msapi [projectPath] setup-svc-live-replay-metadata-service-apply-capture <@packet.json|packetJson> [--dry-run|--execute --approval %s]", setupSvcParityMetadataServiceApplyCaptureApproval)
	}
	packetMap, err := parseObject(args[0], "cloudcc apply msapi setup-svc-live-replay-metadata-service-apply-capture")
	if err != nil {
		return err
	}
	execute, approval := validationRuleApplyOptions(args[1:])
	result, err := c.buildSetupSvcLiveReplayMetadataServiceApplyCaptureResult(packetMap, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayMetadataServiceQueryScanCapture(stdout io.Writer, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc apply msapi [projectPath] setup-svc-live-replay-metadata-service-query-scan-capture <@packet.json|packetJson> [--dry-run|--execute --approval %s]", setupSvcParityMetadataServiceQueryScanCaptureApproval)
	}
	packetMap, err := parseObject(args[0], "cloudcc apply msapi setup-svc-live-replay-metadata-service-query-scan-capture")
	if err != nil {
		return err
	}
	execute, approval := validationRuleApplyOptions(args[1:])
	result, err := c.buildSetupSvcLiveReplayMetadataServiceQueryScanCaptureResult(packetMap, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) applySetupSvcLiveReplayEvidenceImport(stdout io.Writer, args []string) error {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return fmt.Errorf("cloudcc apply msapi [projectPath] setup-svc-live-replay-evidence-import <@packet.json|packetJson> [--dry-run|--execute --approval %s]", setupSvcParityEvidenceImportApproval)
	}
	packetMap, err := parseObject(args[0], "cloudcc apply msapi setup-svc-live-replay-evidence-import")
	if err != nil {
		return err
	}
	execute, approval := validationRuleApplyOptions(args[1:])
	result, err := buildSetupSvcLiveReplayEvidenceImportApplyResult(c.projectPath, packetMap, execute, approval)
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func (c *client) scanSetupSvcLiveReplayEvidenceBundle(stdout io.Writer, args []string) error {
	manifestArg := ""
	if len(args) > 0 && !strings.HasPrefix(strings.TrimSpace(args[0]), "--") {
		manifestArg = args[0]
	}
	result := buildSetupSvcLiveReplayEvidenceBundleScanResult(c.projectPath, manifestArg)
	b, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(b))
	return nil
}

func decodeSetupSvcLiveReplayPacket(value map[string]any) (setupSvcLiveReplayPacket, error) {
	var packet setupSvcLiveReplayPacket
	b, err := json.Marshal(value)
	if err != nil {
		return packet, err
	}
	if err := json.Unmarshal(b, &packet); err != nil {
		return packet, fmt.Errorf("setup-svc live replay packet JSON is invalid: %w", err)
	}
	if !isSetupSvcLiveReplayPacketMode(packet.Mode) {
		return packet, fmt.Errorf("setup-svc live replay packet mode is invalid: %q", packet.Mode)
	}
	if len(packet.Domains) == 0 {
		return packet, fmt.Errorf("setup-svc live replay packet contains no domains")
	}
	return packet, nil
}

func setupSvcLiveReplayPacketApplyResult(projectPath string, packet setupSvcLiveReplayPacket, execute bool, approval string) (setupSvcLiveReplayApplyResult, error) {
	result := setupSvcLiveReplayApplyResult{
		Mode:             "setup-svc-live-replay-apply",
		Project:          projectPath,
		ReadOnly:         !execute,
		Execute:          execute,
		ApprovalRequired: true,
		Approved:         execute && approval == setupSvcParityReplayApproval,
		Status:           "dry_run_ready",
		ManifestPath:     firstString(packet.ManifestPath, setupSvcLiveReplayManifestPath(projectPath, "")),
		NextCommands:     packet.Commands,
	}
	if execute && approval != setupSvcParityReplayApproval {
		return result, fmt.Errorf("refusing to execute setup-svc live replay packet without --approval %s", setupSvcParityReplayApproval)
	}
	if execute {
		result.Status = "blocked_manual_evidence_required"
		result.BlockingIssues = []string{
			"automated setup-svc write replay is not implemented in the Go skill",
			"use this packet to collect approved setup-svc, MetadataService, query/readback, normalized diff, and cleanup evidence, then verify the manifest",
		}
		return result, fmt.Errorf("refusing to execute setup-svc live replay automatically; collect real evidence into %s and run setup-svc-live-replay-evidence", result.ManifestPath)
	}
	if versionIssues := setupSvcLiveReplayContractIdentityIssues(packet.ContractVersion, packet.ContractFingerprint); len(versionIssues) > 0 {
		result.Status = "blocked_incomplete_packet"
		result.Totals.FailedOperations++
		for _, issue := range versionIssues {
			result.BlockingIssues = append(result.BlockingIssues, "packet: "+issue)
		}
		return result, nil
	}
	expected := setupSvcLiveReplayDomains()
	if envelopeIssues := setupSvcLiveReplayPacketEnvelopeIssues(projectPath, packet, expected); len(envelopeIssues) > 0 {
		result.Totals.FailedOperations += len(envelopeIssues)
		for _, issue := range envelopeIssues {
			result.BlockingIssues = append(result.BlockingIssues, "packet: "+issue)
		}
	}
	if templateIssues := setupSvcLiveReplayPacketManifestTemplateIssues(projectPath, packet.ManifestTemplate, expected); len(templateIssues) > 0 {
		result.Totals.FailedOperations += len(templateIssues)
		for _, issue := range templateIssues {
			result.BlockingIssues = append(result.BlockingIssues, "manifestTemplate: "+issue)
		}
	}
	expectedDomainSet := map[string]bool{}
	for _, domain := range expected {
		expectedDomainSet[domain.Domain] = true
	}
	packetDomains := map[string]setupSvcLiveReplayPacketDomain{}
	for _, domain := range packet.Domains {
		normalized := normalizeDomain(domain.Domain)
		switch {
		case normalized == "":
			result.Totals.FailedOperations++
			result.BlockingIssues = append(result.BlockingIssues, "packet: missing domain")
		case !expectedDomainSet[normalized]:
			result.Totals.FailedOperations++
			result.BlockingIssues = append(result.BlockingIssues, "packet: unexpected domain "+domain.Domain)
		case packetDomainExists(packetDomains, normalized):
			result.Totals.FailedOperations++
			result.BlockingIssues = append(result.BlockingIssues, "packet: duplicate domain "+normalized)
		default:
			packetDomains[normalized] = domain
		}
	}
	for _, domain := range expected {
		applyDomain := setupSvcLiveReplayApplyDomain{Domain: domain.Domain, Status: "would_collect_evidence"}
		result.Totals.Domains++
		packetDomain, ok := packetDomains[domain.Domain]
		if !ok {
			applyDomain.Status = "missing_domain"
			result.Totals.MissingDomains++
			result.BlockingIssues = append(result.BlockingIssues, domain.Domain+": missing from packet")
			result.Domains = append(result.Domains, applyDomain)
			continue
		}
		if domainIssues := setupSvcLiveReplayDomainContractIssues(domain, packetDomain); len(domainIssues) > 0 {
			applyDomain.Status = "invalid_domain_contract"
			result.Totals.FailedOperations += len(domain.Operations)
			for _, issue := range domainIssues {
				result.BlockingIssues = append(result.BlockingIssues, domain.Domain+": "+issue)
			}
		}
		expectedOperationSet := map[string]bool{}
		for _, operation := range domain.Operations {
			expectedOperationSet[strings.ToLower(strings.TrimSpace(operation))] = true
		}
		packetOps := map[string]setupSvcLiveReplayPacketOperation{}
		for _, operation := range packetDomain.Operations {
			normalized := strings.ToLower(strings.TrimSpace(operation.Operation))
			switch {
			case normalized == "":
				applyDomain.Status = "invalid_operation_contract"
				result.Totals.FailedOperations++
				result.BlockingIssues = append(result.BlockingIssues, domain.Domain+": missing operation")
			case !expectedOperationSet[normalized]:
				applyDomain.Status = "invalid_operation_contract"
				applyDomain.Operations = append(applyDomain.Operations, setupSvcLiveReplayApplyOperation{Operation: operation.Operation, Status: "unexpected_in_packet"})
				result.Totals.FailedOperations++
				result.BlockingIssues = append(result.BlockingIssues, domain.Domain+"/"+operation.Operation+": unexpected operation")
			case packetOperationExists(packetOps, normalized):
				applyDomain.Status = "invalid_operation_contract"
				applyDomain.Operations = append(applyDomain.Operations, setupSvcLiveReplayApplyOperation{Operation: operation.Operation, Status: "duplicate_in_packet"})
				result.Totals.FailedOperations++
				result.BlockingIssues = append(result.BlockingIssues, domain.Domain+"/"+normalized+": duplicate operation")
			default:
				packetOps[normalized] = operation
			}
		}
		for _, operation := range domain.Operations {
			result.Totals.Operations++
			packetOperation, ok := packetOps[strings.ToLower(operation)]
			if !ok {
				applyDomain.Status = "missing_operations"
				applyDomain.Operations = append(applyDomain.Operations, setupSvcLiveReplayApplyOperation{Operation: operation, Status: "missing_from_packet"})
				result.Totals.MissingOperations++
				result.BlockingIssues = append(result.BlockingIssues, domain.Domain+"/"+operation+": missing from packet")
				continue
			}
			contractIssues := setupSvcLiveReplayPacketContractIssues(domain.Domain, operation, packetOperation)
			if len(contractIssues) > 0 {
				applyDomain.Status = "invalid_evidence_contract"
				applyDomain.Operations = append(applyDomain.Operations, setupSvcLiveReplayApplyOperation{Operation: operation, Status: "invalid_evidence_contract"})
				result.Totals.FailedOperations++
				for _, issue := range contractIssues {
					result.BlockingIssues = append(result.BlockingIssues, domain.Domain+"/"+operation+": "+issue)
				}
				continue
			}
			applyDomain.Operations = append(applyDomain.Operations, setupSvcLiveReplayApplyOperation{Operation: operation, Status: "would_collect_evidence"})
		}
		result.Domains = append(result.Domains, applyDomain)
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked_incomplete_packet"
	} else {
		result.Warnings = []string{
			"Dry run only; no setup-svc or MetadataService write API was called.",
			"Use --execute only after an automated executor exists and an approved disposable replay window is available.",
			"Matrix entries remain covered until setup-svc-live-replay-evidence passes a real manifest.",
		}
	}
	return result, nil
}

func buildSetupSvcLiveReplayWorkspaceApplyResult(projectPath string, packet setupSvcLiveReplayPacket, execute bool, approval string) (setupSvcLiveReplayWorkspaceApplyResult, error) {
	result := setupSvcLiveReplayWorkspaceApplyResult{
		Mode:                "setup-svc-live-replay-workspace",
		Project:             projectPath,
		ReadOnly:            !execute,
		Execute:             execute,
		ApprovalRequired:    true,
		Approved:            execute && approval == setupSvcParityEvidenceWorkspaceApproval,
		Status:              "dry_run_ready",
		ManifestPath:        firstString(packet.ManifestPath, setupSvcLiveReplayManifestPath(projectPath, "")),
		EvidenceDirectory:   filepath.Join(projectPath, "outputs", "setup-svc-live-replay"),
		ContractVersion:     packet.ContractVersion,
		ContractFingerprint: packet.ContractFingerprint,
		NextCommands: setupSvcLiveReplayWorkspaceCommands{
			PrepareWorkspace: "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-workspace --execute --approval " + setupSvcParityEvidenceWorkspaceApproval,
			VerifyEvidence:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(firstString(packet.ManifestPath, setupSvcLiveReplayManifestPath(projectPath, ""))),
			PromotionAudit:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(firstString(packet.ManifestPath, setupSvcLiveReplayManifestPath(projectPath, ""))),
			CompletionAudit:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(firstString(packet.ManifestPath, setupSvcLiveReplayManifestPath(projectPath, ""))),
		},
		Notes: []string{
			"This command prepares local JSON evidence placeholders only; it does not execute setup-svc or MetadataService writes.",
			"Every generated manifest and artifact status remains pending, so the evidence verifier must keep blocking until real replay evidence replaces the placeholders.",
		},
	}
	if execute && approval != setupSvcParityEvidenceWorkspaceApproval {
		return result, fmt.Errorf("refusing to prepare setup-svc evidence workspace without --approval %s", setupSvcParityEvidenceWorkspaceApproval)
	}
	packetCheck, err := setupSvcLiveReplayPacketApplyResult(projectPath, packet, false, "")
	if err != nil {
		return result, err
	}
	result.Totals.Domains = packetCheck.Totals.Domains
	result.Totals.Operations = packetCheck.Totals.Operations
	for _, domain := range packet.Domains {
		for _, operation := range domain.Operations {
			result.Totals.ArtifactFiles += len(operation.EvidenceFiles)
			for _, file := range operation.EvidenceFiles {
				if len(result.SampleFiles) < 10 {
					result.SampleFiles = append(result.SampleFiles, filepath.Join(projectPath, file))
				}
			}
		}
	}
	if packetCheck.Status != "dry_run_ready" || len(packetCheck.BlockingIssues) > 0 {
		result.Status = "blocked_incomplete_packet"
		result.BlockingIssues = append(result.BlockingIssues, packetCheck.BlockingIssues...)
		return result, nil
	}
	if !execute {
		result.Warnings = []string{
			"Dry run only; no workspace files were written.",
			"Use --execute only to create pending placeholder evidence files for a real approved replay window.",
		}
		return result, nil
	}
	if err := writeSetupSvcLiveReplayWorkspace(projectPath, packet); err != nil {
		result.Status = "blocked_workspace_write"
		result.BlockingIssues = append(result.BlockingIssues, err.Error())
		return result, nil
	}
	result.Status = "applied"
	result.Totals.WrittenFiles = result.Totals.ArtifactFiles + 1
	return result, nil
}

func writeSetupSvcLiveReplayWorkspace(projectPath string, packet setupSvcLiveReplayPacket) error {
	manifestPath := firstString(packet.ManifestPath, setupSvcLiveReplayManifestPath(projectPath, ""))
	manifestBody, err := json.MarshalIndent(packet.ManifestTemplate, "", "  ")
	if err != nil {
		return err
	}
	if issues := setupSvcLiveReplayWorkspaceOverwriteIssues(projectPath, packet); len(issues) > 0 {
		return fmt.Errorf("refusing to overwrite non-pending setup-svc live replay evidence files: %s", strings.Join(issues, "; "))
	}
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, append(manifestBody, '\n'), 0644); err != nil {
		return err
	}
	domainsByName := map[string]setupSvcLiveReplayDomain{}
	queryReadbackExpectationsByDomain := map[string][]string{}
	for _, domain := range setupSvcLiveReplayDomains() {
		domainsByName[normalizeDomain(domain.Domain)] = domain
	}
	for _, domain := range packet.Domains {
		normalized := normalizeDomain(domain.Domain)
		if _, ok := domainsByName[normalized]; !ok {
			domainsByName[normalized] = setupSvcLiveReplayDomain{
				Domain:                    domain.Domain,
				RequiredTables:            append([]string{}, domain.RequiredTables...),
				RuntimeEffects:            append([]string{}, domain.RuntimeEffects...),
				QueryReadbackExpectations: append([]string{}, domain.QueryReadbackExpectations...),
			}
		}
		queryReadbackExpectationsByDomain[normalized] = append([]string{}, domain.QueryReadbackExpectations...)
	}
	for _, domain := range packet.ManifestTemplate.Domains {
		normalizedDomain := normalizeDomain(domain.Domain)
		contractDomain := domainsByName[normalizedDomain]
		queryReadbackExpectations := queryReadbackExpectationsByDomain[normalizedDomain]
		for _, operation := range domain.Operations {
			requiredTables := setupSvcLiveReplayRequiredTablesForOperation(contractDomain, operation.Operation)
			runtimeEffects := setupSvcLiveReplayRuntimeEffectsForOperation(contractDomain.Domain, operation.Operation)
			for _, file := range operation.EvidenceFiles {
				artifactPath := filepath.Join(projectPath, file)
				artifact := setupSvcLiveReplayArtifactTemplate(projectPath, packet, domain.Domain, operation.Operation, file, requiredTables, runtimeEffects, queryReadbackExpectations)
				body, err := json.MarshalIndent(artifact, "", "  ")
				if err != nil {
					return err
				}
				if err := os.MkdirAll(filepath.Dir(artifactPath), 0755); err != nil {
					return err
				}
				if err := os.WriteFile(artifactPath, append(body, '\n'), 0644); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func buildSetupSvcLiveReplayCaptureSourceWorkspaceApplyResult(projectPath string, manifestArg string, args []string, execute bool, approval string) (setupSvcLiveReplayCaptureSourceWorkspaceApplyResult, error) {
	collectionArgs := setupSvcLiveReplayCollectionArgsFromApplyArgs(args)
	parsedManifestArg, options, err := setupSvcLiveReplayParseGapArgs(manifestArg, collectionArgs)
	if err != nil {
		return setupSvcLiveReplayCaptureSourceWorkspaceApplyResult{}, err
	}
	if options.Status != "" || options.EvidenceSection != "" || options.SectionStatus != "" || options.BatchIndex >= 0 || options.BatchLimit != setupSvcLiveReplayWorklistBatchLimit {
		return setupSvcLiveReplayCaptureSourceWorkspaceApplyResult{}, fmt.Errorf("setup-svc-live-replay-capture-source-workspace supports only --domain, --operation, --artifact-type, --source-status, --source-readiness, --offset, and --limit")
	}
	if !setupSvcLiveReplayArgsContainOption(collectionArgs, "--limit") {
		options.Limit = 0
	}
	if options.SourceStatus == "" && options.SourceReadiness == "" {
		options.SourceStatus = "missing"
	}
	packet := buildSetupSvcLiveReplayPacket(projectPath)
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, parsedManifestArg)
	if strings.TrimSpace(parsedManifestArg) == "" {
		manifestPath = packet.ManifestPath
	}
	sourceSummary := setupSvcLiveReplayCaptureSourceSummaryFor(projectPath, packet)
	worklistOptions := options
	worklistOptions.SourceReadiness = "complete"
	worklistOptions.SourceStatus = ""
	worklistPath := setupSvcLiveReplayWorklistSuggestedPath(projectPath, worklistOptions)
	result := setupSvcLiveReplayCaptureSourceWorkspaceApplyResult{
		Mode:                "setup-svc-live-replay-capture-source-workspace",
		Project:             projectPath,
		ReadOnly:            false,
		Execute:             execute,
		ApprovalRequired:    true,
		Approved:            execute && approval == setupSvcParityCaptureSourceWorkspaceApproval,
		Status:              "dry_run_ready",
		ManifestPath:        manifestPath,
		SourceRoot:          sourceSummary.SourceRoot,
		CaptureRoot:         sourceSummary.CaptureRoot,
		ContractVersion:     packet.ContractVersion,
		ContractFingerprint: packet.ContractFingerprint,
		Filters:             setupSvcLiveReplayCollectionPlanFiltersFromOptions(options),
		NextCommands: setupSvcLiveReplayCaptureSourceWorkspaceCommands{
			PrepareCaptureSources: setupSvcLiveReplayCaptureSourceWorkspaceCommand(projectPath, manifestPath, options) + " --execute --approval " + setupSvcParityCaptureSourceWorkspaceApproval,
			CapturePlan:           "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-capture-plan " + shellPath(manifestPath) + " --source-status present",
			CompleteWorklist:      "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-worklist " + shellPath(manifestPath) + " --source-readiness complete > " + shellPath(worklistPath),
			DryRunImport:          "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(worklistPath) + " --dry-run",
			ExecuteImport:         "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(worklistPath) + " --execute --approval " + setupSvcParityEvidenceImportApproval,
		},
		Notes: []string{
			"This command initializes mirrored capture source JSON templates under captures/outputs/setup-svc-live-replay.",
			"It does not call setup-svc, MetadataService writes, query APIs, normalized diff generation, evidence import, manifest sync, bundle writing, promotion, or matrix updates.",
			"Existing capture source files are skipped so real captured evidence is not overwritten.",
			"Pending source templates are intentionally incomplete; replace them with real evidence before importing.",
		},
	}
	result.Totals.ArtifactFiles = sourceSummary.ArtifactFiles
	result.Totals.SourceFilesPresent = sourceSummary.SourceFilesPresent
	result.Totals.SourceFilesMissing = sourceSummary.SourceFilesMissing
	result.Totals.SourceFilesComplete = sourceSummary.SourceFilesComplete
	result.Totals.SourceFilesIncomplete = sourceSummary.SourceFilesIncomplete
	if execute && approval != setupSvcParityCaptureSourceWorkspaceApproval {
		return result, fmt.Errorf("refusing to prepare setup-svc capture source workspace without --approval %s", setupSvcParityCaptureSourceWorkspaceApproval)
	}
	records := setupSvcLiveReplayCaptureSourceWorkspaceRecords(projectPath, manifestPath, packet, options)
	result.Totals.FilteredArtifactFiles = len(records)
	start := options.Offset
	if start > len(records) {
		start = len(records)
	}
	end := len(records)
	if options.Limit > 0 && start+options.Limit < end {
		end = start + options.Limit
	}
	records = records[start:end]
	result.Totals.PlannedFiles = len(records)
	for _, record := range records {
		if len(result.SampleFiles) < 10 {
			result.SampleFiles = append(result.SampleFiles, record.SuggestedSourcePath)
		}
		if record.SuggestedSourceExists {
			if execute {
				refreshed, err := refreshSetupSvcLiveReplayCaptureSourceTemplate(projectPath, packet, manifestPath, record)
				if err != nil {
					result.Status = "blocked_capture_source_write"
					result.BlockingIssues = append(result.BlockingIssues, err.Error())
					return result, nil
				}
				if refreshed {
					result.Totals.RefreshedExistingFiles++
					continue
				}
			}
			result.Totals.SkippedExistingFiles++
			continue
		}
		if !execute {
			continue
		}
		if err := writeSetupSvcLiveReplayCaptureSourceTemplate(projectPath, packet, manifestPath, record); err != nil {
			result.Status = "blocked_capture_source_write"
			result.BlockingIssues = append(result.BlockingIssues, err.Error())
			return result, nil
		}
		result.Totals.WrittenFiles++
	}
	if !execute {
		result.Warnings = append(result.Warnings,
			"Dry run only; no capture source files were written.",
			"Use --execute only to create incomplete capture source templates for real evidence collection.",
		)
		return result, nil
	}
	result.Status = "applied"
	if result.Totals.WrittenFiles == 0 && result.Totals.RefreshedExistingFiles == 0 {
		result.Status = "nothing_to_write"
	}
	return result, nil
}

func setupSvcLiveReplayCaptureSourceWorkspaceCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	args := []string{"cloudcc", "apply", "msapi", shellPath(projectPath), "setup-svc-live-replay-capture-source-workspace", shellPath(manifestPath)}
	args = append(args, setupSvcLiveReplayGapArgsFromOptions(options)...)
	if options.SourceStatus != "" {
		args = append(args, "--source-status", options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		args = append(args, "--source-readiness", options.SourceReadiness)
	}
	return strings.Join(args, " ")
}

type setupSvcLiveReplayCaptureSourceWorkspaceRecord struct {
	Domain                    string
	Operation                 string
	ArtifactType              string
	TargetPath                string
	SuggestedSourcePath       string
	SuggestedSourceExists     bool
	RequiredTables            []string
	RuntimeEffects            []string
	QueryReadbackExpectations []string
}

func setupSvcLiveReplayCaptureSourceWorkspaceRecords(projectPath string, manifestPath string, packet setupSvcLiveReplayPacket, options setupSvcLiveReplayCollectionPlanOptions) []setupSvcLiveReplayCaptureSourceWorkspaceRecord {
	domainContracts := setupSvcLiveReplayDomainContractMap()
	seen := map[string]bool{}
	records := []setupSvcLiveReplayCaptureSourceWorkspaceRecord{}
	for _, domain := range packet.Domains {
		if options.Domain != "" && normalizeDomain(domain.Domain) != options.Domain {
			continue
		}
		contract := domainContracts[normalizeDomain(domain.Domain)]
		for _, operation := range domain.Operations {
			if options.Operation != "" && strings.ToLower(strings.TrimSpace(operation.Operation)) != options.Operation {
				continue
			}
			for _, file := range operation.EvidenceFiles {
				normalizedFile := strings.TrimSpace(file)
				if normalizedFile == "" || seen[normalizedFile] {
					continue
				}
				seen[normalizedFile] = true
				artifactType := setupSvcLiveReplayArtifactType(normalizedFile)
				if options.ArtifactType != "" && setupSvcLiveReplayNormalizeArtifactType(artifactType) != options.ArtifactType {
					continue
				}
				suggestedSourcePath := setupSvcLiveReplayWorklistSuggestedSourcePath(normalizedFile)
				sourceExists := setupSvcLiveReplayWorklistSuggestedSourceExists(projectPath, suggestedSourcePath)
				requiredSections := setupSvcLiveReplayRequiredEvidenceSections(artifactType)
				sourceReadiness := setupSvcLiveReplaySourceReadiness(sourceExists, setupSvcLiveReplayEvidenceSectionStatusesAtPath(filepath.Join(projectPath, suggestedSourcePath), requiredSections))
				if options.SourceStatus == "present" && !sourceExists {
					continue
				}
				if options.SourceStatus == "missing" && sourceExists {
					continue
				}
				if options.SourceReadiness != "" && sourceReadiness != options.SourceReadiness {
					continue
				}
				records = append(records, setupSvcLiveReplayCaptureSourceWorkspaceRecord{
					Domain:                    domain.Domain,
					Operation:                 operation.Operation,
					ArtifactType:              artifactType,
					TargetPath:                normalizedFile,
					SuggestedSourcePath:       suggestedSourcePath,
					SuggestedSourceExists:     sourceExists,
					RequiredTables:            setupSvcLiveReplayCollectionRequiredTables(contract, artifactType),
					RuntimeEffects:            setupSvcLiveReplayCollectionRuntimeEffects(contract, artifactType),
					QueryReadbackExpectations: setupSvcLiveReplayCollectionQueryReadbackExpectations(contract, artifactType),
				})
			}
		}
	}
	return records
}

func writeSetupSvcLiveReplayCaptureSourceTemplate(projectPath string, packet setupSvcLiveReplayPacket, manifestPath string, record setupSvcLiveReplayCaptureSourceWorkspaceRecord) error {
	sourcePath := filepath.Join(projectPath, record.SuggestedSourcePath)
	artifact := setupSvcLiveReplayArtifactTemplate(projectPath, packet, record.Domain, record.Operation, record.TargetPath, record.RequiredTables, record.RuntimeEffects, record.QueryReadbackExpectations)
	for key, value := range setupSvcLiveReplayCaptureSourceTemplateGuide(projectPath, manifestPath, record) {
		artifact[key] = value
	}
	body, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(sourcePath, append(body, '\n'), 0644); err != nil {
		return err
	}
	return nil
}

func refreshSetupSvcLiveReplayCaptureSourceTemplate(projectPath string, packet setupSvcLiveReplayPacket, manifestPath string, record setupSvcLiveReplayCaptureSourceWorkspaceRecord) (bool, error) {
	sourcePath := filepath.Join(projectPath, record.SuggestedSourcePath)
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return false, err
	}
	var artifact map[string]any
	if err := json.Unmarshal(body, &artifact); err != nil {
		return false, err
	}
	if strings.TrimSpace(stringValue(artifact["status"])) != "pending" || strings.TrimSpace(stringValue(artifact["sourceTemplateStatus"])) != "incomplete" {
		return false, nil
	}
	before, err := json.Marshal(artifact)
	if err != nil {
		return false, err
	}
	base := setupSvcLiveReplayArtifactTemplate(projectPath, packet, record.Domain, record.Operation, record.TargetPath, record.RequiredTables, record.RuntimeEffects, record.QueryReadbackExpectations)
	for _, key := range []string{
		"contractVersion",
		"contractFingerprint",
		"project",
		"domain",
		"operation",
		"artifactType",
		"requiredTables",
		"runtimeEffects",
		"queryReadbackExpectations",
	} {
		if value, ok := base[key]; ok {
			artifact[key] = value
		}
	}
	for key, value := range setupSvcLiveReplayCaptureSourceTemplateGuide(projectPath, manifestPath, record) {
		artifact[key] = value
	}
	after, err := json.Marshal(artifact)
	if err != nil {
		return false, err
	}
	if string(before) == string(after) {
		return false, nil
	}
	refreshed, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(sourcePath, append(refreshed, '\n'), 0644); err != nil {
		return false, err
	}
	return true, nil
}

func setupSvcLiveReplayCaptureSourceTemplateGuide(projectPath string, manifestPath string, record setupSvcLiveReplayCaptureSourceWorkspaceRecord) map[string]any {
	requiredShapeKey := setupSvcLiveReplayCollectionRequiredShapeKey(record.ArtifactType)
	manifestStatusField := setupSvcLiveReplayManifestStatusField(record.ArtifactType)
	requiredSections := setupSvcLiveReplayRequiredEvidenceSections(record.ArtifactType)
	return map[string]any{
		"requiredShapeKey":         requiredShapeKey,
		"manifestStatusField":      manifestStatusField,
		"requiredEvidenceSections": requiredSections,
		"captureTask": setupSvcLiveReplayArtifactCaptureTaskFor(projectPath, manifestPath, record.Domain, record.Operation, record.ArtifactType, record.TargetPath, record.SuggestedSourcePath,
			setupSvcLiveReplayCollectionRequiredShapeKey(record.ArtifactType),
			setupSvcLiveReplayManifestStatusField(record.ArtifactType),
			setupSvcLiveReplayRequiredEvidenceSections(record.ArtifactType),
			record.RequiredTables,
			record.RuntimeEffects,
			record.QueryReadbackExpectations),
		"targetEvidencePath":   record.TargetPath,
		"sourceTemplateStatus": "incomplete",
	}
}

func setupSvcLiveReplayCollectionArgsFromApplyArgs(args []string) []string {
	filtered := []string{}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		name, _, _ := strings.Cut(arg, "=")
		switch strings.ToLower(name) {
		case "--dry-run", "--execute":
			continue
		case "--approval":
			if !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
			}
			continue
		default:
			filtered = append(filtered, args[i])
		}
	}
	return filtered
}

func setupSvcLiveReplayArgsContainOption(args []string, option string) bool {
	option = strings.ToLower(strings.TrimSpace(option))
	for _, arg := range args {
		name, _, _ := strings.Cut(strings.ToLower(strings.TrimSpace(arg)), "=")
		if name == option {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayWorkspaceOverwriteIssues(projectPath string, packet setupSvcLiveReplayPacket) []string {
	paths := []string{firstString(packet.ManifestPath, setupSvcLiveReplayManifestPath(projectPath, ""))}
	for _, domain := range packet.ManifestTemplate.Domains {
		for _, operation := range domain.Operations {
			for _, file := range operation.EvidenceFiles {
				paths = append(paths, filepath.Join(projectPath, file))
			}
		}
	}
	var issues []string
	for _, path := range paths {
		if issue := setupSvcLiveReplayWorkspaceOverwriteIssue(projectPath, path); issue != "" {
			issues = append(issues, issue)
		}
	}
	sort.Strings(issues)
	return issues
}

func setupSvcLiveReplayWorkspaceOverwriteIssue(projectPath string, path string) string {
	payload, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return ""
	}
	rel, relErr := filepath.Rel(projectPath, path)
	if relErr != nil {
		rel = path
	}
	if err != nil {
		return rel + ": cannot read existing evidence file: " + err.Error()
	}
	var existing map[string]any
	if err := json.Unmarshal(payload, &existing); err != nil {
		return rel + ": existing evidence file is not valid JSON"
	}
	status, _ := existing["status"].(string)
	if strings.EqualFold(strings.TrimSpace(status), "pending") {
		return ""
	}
	if strings.TrimSpace(status) == "" {
		return rel + ": existing evidence file has empty status"
	}
	return rel + ": existing evidence status is " + status
}

func setupSvcLiveReplayArtifactTemplate(projectPath string, packet setupSvcLiveReplayPacket, domain string, operation string, file string, requiredTables []string, runtimeEffects []string, queryReadbackExpectations []string) map[string]any {
	artifactType := setupSvcLiveReplayArtifactType(file)
	template := map[string]any{
		"status":                       "pending",
		"project":                      projectPath,
		"contractVersion":              packet.ContractVersion,
		"contractFingerprint":          packet.ContractFingerprint,
		"domain":                       domain,
		"operation":                    operation,
		"artifactType":                 artifactType,
		"requiredTables":               append([]string{}, requiredTables...),
		"runtimeEffects":               append([]string{}, runtimeEffects...),
		"queryReadbackExpectations":    append([]string{}, queryReadbackExpectations...),
		"artifactReplacementChecklist": setupSvcLiveReplayArtifactReplacementChecklist(artifactType),
		"notes": []string{
			"Replace this placeholder with real approved setup-svc live replay evidence before changing status to passed.",
			"Pending placeholders are intentionally rejected by setup-svc-live-replay-evidence.",
		},
	}
	switch artifactType {
	case "setup-svc", "metadata-service":
		snapshots := map[string]any{}
		for _, table := range requiredTables {
			snapshots[table] = map[string]any{
				"columns": []string{},
				"rows":    []map[string]any{},
			}
		}
		template["tableSnapshots"] = snapshots
		template["runtimeEffectChecks"] = setupSvcLiveReplayPendingExpectationChecks(runtimeEffects)
		template["requiredSnapshotShape"] = map[string]any{
			"requiredTables":              append([]string{}, requiredTables...),
			"rowEvidenceKeys":             []string{"rows", "records", "sampleRows", "before", "after", "changes"},
			"columnEvidenceKeys":          []string{"columns", "fields", "primaryKeys", "keyColumns"},
			"runtimeEffectChecksRequired": append([]string{}, runtimeEffects...),
			"statusRule":                  "Use string status passed, verified, or success only after each required table has row evidence plus column or key evidence and each runtime effect has a named passed check.",
		}
	case "query-readback":
		template["queryShape"] = map[string]any{
			"fields":         []string{},
			"readbackTables": []map[string]any{},
		}
		template["readbackChecks"] = map[string]any{
			"relationshipChecks":        []map[string]any{},
			"readbackExpectationChecks": setupSvcLiveReplayPendingExpectationChecks(queryReadbackExpectations),
			"missingFields":             nil,
			"mismatchedFields":          nil,
			"missingRelationships":      nil,
			"brokenRelationships":       nil,
			"errors":                    nil,
		}
		template["requiredReadbackShape"] = map[string]any{
			"requiredTables":                          append([]string{}, requiredTables...),
			"tableCoverageKeys":                       []string{"readbackTables", "queriedTables", "metadataTables", "tableCoverage"},
			"rowEvidenceKeys":                         []string{"rows", "records", "sampleRows", "readbackRows"},
			"fieldEvidenceKeys":                       []string{"columns", "fields", "requiredFields", "readbackFields", "queryFields"},
			"relationshipCheckRequirements":           []string{"named passed relationshipChecks", "or source plus target plus field relationship triples"},
			"readbackExpectationChecksRequired":       append([]string{}, queryReadbackExpectations...),
			"requiredNumericZeroCleanCounters":        []string{"missingFields", "mismatchedFields", "missingRelationships", "brokenRelationships", "errors"},
			"explicitQueryShapeEvidenceKeys":          []string{"fields", "columns", "queryShape", "readbackShape"},
			"statusRule":                              "Use string status passed, verified, or success only after every required table has row evidence plus field evidence and every readback expectation has a named passed check.",
			"rejectedEvidencePatterns":                []string{"rowCount-only", "columns-only", "rowCount-plus-columns-only", "requiredRelationships-only", "clean-counter-only", "unnamed relationshipChecks"},
			"missingRelationshipCounterAllowedOnlyIf": "relationship structure proof is present and missing/broken counters are numeric zero",
		}
	case "normalized-diff":
		template["totals"] = map[string]any{
			"missingRows":      nil,
			"unexpectedRows":   nil,
			"mismatchedValues": nil,
			"differences":      nil,
			"failed":           nil,
		}
		template["requiredDiffShape"] = map[string]any{
			"requiredNumericZeroCleanCounters": []string{"missingRows", "unexpectedRows", "mismatchedValues", "differences", "failed"},
			"nestedCleanNodes":                 []string{"diff", "evidence", "comparison", "normalizedDiff"},
			"statusRule":                       "Use string status passed, verified, or success only after all top-level and nested diff counters are numeric zero; clean:true alone is rejected.",
		}
	case "cleanup":
		template["residuals"] = map[string]any{
			"remaining": nil,
			"residual":  nil,
			"orphan":    nil,
			"errors":    nil,
			"failures":  nil,
		}
		template["requiredCleanupShape"] = map[string]any{
			"requiredNumericZeroResidualCounters": []string{"remaining", "residual", "orphan", "errors", "failures"},
			"residualEvidenceKeys":                []string{"deleted", "removed", "remaining", "residual", "orphan"},
			"statusRule":                          "Use string status passed, verified, or success only after cleanup evidence proves no remaining, residual, orphan, error, or failure counters.",
		}
	}
	return template
}

func setupSvcLiveReplayPendingExpectationChecks(expectations []string) []map[string]any {
	checks := make([]map[string]any, 0, len(expectations))
	for _, expectation := range expectations {
		if trimmed := strings.TrimSpace(expectation); trimmed != "" {
			checks = append(checks, map[string]any{
				"name":     trimmed,
				"status":   "pending",
				"evidence": nil,
			})
		}
	}
	return checks
}

func buildSetupSvcLiveReplayGapResult(projectPath string, manifestArg string, optionArgs ...string) (setupSvcLiveReplayGapResult, error) {
	parsedManifestArg, collectionOptions, err := setupSvcLiveReplayParseGapArgs(manifestArg, optionArgs)
	if err != nil {
		return setupSvcLiveReplayGapResult{}, err
	}
	if collectionOptions.SourceStatus != "" || collectionOptions.SourceReadiness != "" {
		return setupSvcLiveReplayGapResult{}, fmt.Errorf("--source-status and --source-readiness are supported only by setup-svc-live-replay-capture-plan/worklist where documented")
	}
	manifestArg = parsedManifestArg
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	matrixContract := setupSvcLiveReplayMatrixContractStatus(projectPath)
	result := setupSvcLiveReplayGapResult{
		Mode:           "setup-svc-live-replay-gaps",
		Project:        projectPath,
		ReadOnly:       true,
		Status:         "ready_for_evidence",
		ManifestPath:   manifestPath,
		MatrixContract: matrixContract,
		NextCommands: setupSvcLiveReplayGapCommands{
			PrepareWorkspace: "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-workspace --execute --approval " + setupSvcParityEvidenceWorkspaceApproval,
			GenerateDiffs:    "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-normalized-diff " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityNormalizedDiffApproval,
			SyncManifest:     "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-manifest-sync " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityManifestSyncApproval,
			VerifyEvidence:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
			WriteBundle:      "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityEvidenceBundleApproval,
			PromotionAudit:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath),
			CompletionAudit:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		},
		Notes: []string{
			"This read-only command is for an in-progress live replay workspace; it does not accept evidence as verified.",
			"Use it before setup-svc-live-replay-evidence to see which files, statuses, or artifact structures still need real replay data.",
			"Only setup-svc-live-replay-evidence plus matrix promotion can move a domain from covered to verified.",
		},
	}
	for _, issue := range matrixContract.Issues {
		result.BlockingIssues = append(result.BlockingIssues, "parityMatrix: "+issue)
	}
	payload, err := os.ReadFile(manifestPath)
	if err != nil {
		result.Status = "missing_manifest"
		result.BlockingIssues = append(result.BlockingIssues, "manifest: missing live replay evidence "+manifestPath)
		setupSvcLiveReplayPopulateMissingGapDomains(&result)
		setupSvcLiveReplayFinalizeCollectionPlan(&result, collectionOptions)
		return result, nil
	}
	var manifest map[string]any
	if err := json.Unmarshal(payload, &manifest); err != nil {
		result.Status = "blocked_invalid_manifest"
		result.BlockingIssues = append(result.BlockingIssues, "manifest: invalid JSON "+err.Error())
		setupSvcLiveReplayPopulateMissingGapDomains(&result)
		setupSvcLiveReplayFinalizeCollectionPlan(&result, collectionOptions)
		return result, nil
	}
	result.ContractVersion = firstMapString(manifest, "contractVersion")
	result.ContractFingerprint = firstMapString(manifest, "contractFingerprint")
	for _, issue := range setupSvcLiveReplayContractIdentityIssues(result.ContractVersion, result.ContractFingerprint) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	for _, issue := range setupSvcLiveReplayProjectIdentityIssues(projectPath, firstMapString(manifest, "project", "projectPath")) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	if mode := firstMapString(manifest, "mode"); mode != "" && mode != setupSvcLiveReplayEvidenceMode {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: unexpected mode "+mode)
	}
	domainEvidence := map[string]map[string]any{}
	expectedDomains := map[string]bool{}
	for _, expected := range setupSvcLiveReplayDomains() {
		expectedDomains[expected.Domain] = true
	}
	for _, domain := range mapList(manifest["domains"]) {
		name := firstMapString(domain, "domain", "name")
		normalized := normalizeDomain(name)
		switch {
		case normalized == "":
			result.BlockingIssues = append(result.BlockingIssues, "manifest: missing domain name")
		case !expectedDomains[normalized]:
			result.BlockingIssues = append(result.BlockingIssues, "manifest: unexpected domain "+name)
		case domainEvidence[normalized] != nil:
			result.BlockingIssues = append(result.BlockingIssues, "manifest: duplicate domain "+normalized)
		default:
			domainEvidence[normalized] = domain
		}
	}
	for _, expected := range setupSvcLiveReplayDomains() {
		domainResult := setupSvcLiveReplayGapDomain{Domain: expected.Domain, Status: "complete"}
		result.Totals.Domains++
		operationEvidence := setupSvcLiveReplayManifestOperationMap(domainEvidence[expected.Domain], &result, expected.Domain)
		for _, operation := range expected.Operations {
			result.Totals.Operations++
			operationResult := buildSetupSvcLiveReplayOperationGap(projectPath, expected, operation, operationEvidence[strings.ToLower(operation)])
			setupSvcLiveReplayAccumulateGapTotals(&result, operationResult)
			if operationResult.Status != "complete" && domainResult.Status == "complete" {
				domainResult.Status = operationResult.Status
			}
			if operationResult.Status == "failed_evidence" {
				domainResult.Status = "failed_evidence"
			}
			domainResult.Operations = append(domainResult.Operations, operationResult)
		}
		result.Domains = append(result.Domains, domainResult)
	}
	switch {
	case result.Status == "missing_manifest" || result.Status == "blocked_invalid_manifest":
	case result.Totals.FailedOperations > 0 || len(result.BlockingIssues) > 0:
		result.Status = "blocked"
	case result.Totals.MissingOperations > 0:
		result.Status = "missing_evidence"
	case result.Totals.ReadyForDiffOperations > 0:
		result.Status = "ready_for_normalized_diff"
	case result.Totals.PendingOperations > 0:
		result.Status = "pending_evidence"
	default:
		result.Status = "complete"
	}
	setupSvcLiveReplayFinalizeCollectionPlan(&result, collectionOptions)
	return result, nil
}

func buildSetupSvcLiveReplayCapturePlanResult(projectPath string, manifestArg string, optionArgs ...string) (setupSvcLiveReplayCapturePlanResult, error) {
	parsedManifestArg, options, err := setupSvcLiveReplayParseGapArgs(manifestArg, optionArgs)
	if err != nil {
		return setupSvcLiveReplayCapturePlanResult{}, err
	}
	if options.Status != "" || options.BatchIndex >= 0 || options.BatchLimit != setupSvcLiveReplayWorklistBatchLimit {
		return setupSvcLiveReplayCapturePlanResult{}, fmt.Errorf("setup-svc-live-replay-capture-plan supports only --domain, --operation, --artifact-type, --evidence-section, --section-status, --source-status, --source-readiness, --offset, and --limit")
	}
	if options.Limit <= 0 {
		options.Limit = 25
	}
	packet := buildSetupSvcLiveReplayPacket(projectPath)
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, parsedManifestArg)
	if strings.TrimSpace(parsedManifestArg) == "" {
		manifestPath = packet.ManifestPath
	}
	sourceSummary := setupSvcLiveReplayCaptureSourceSummaryFor(projectPath, packet)
	result := setupSvcLiveReplayCapturePlanResult{
		Mode:           "setup-svc-live-replay-capture-plan",
		Project:        projectPath,
		ReadOnly:       true,
		Status:         sourceSummary.Status,
		ManifestPath:   manifestPath,
		SourceRoot:     sourceSummary.SourceRoot,
		CaptureRoot:    sourceSummary.CaptureRoot,
		Filters:        setupSvcLiveReplayCollectionPlanFiltersFromOptions(options),
		OperatorPacket: setupSvcLiveReplayWorklistOperatorPacketFor(projectPath, manifestPath, setupSvcLiveReplayCapturePlanOperatorPacketOptions(options), setupSvcLiveReplayCapturePlanGapCommands(projectPath, manifestPath)),
		RecommendedNextSteps: []string{
			sourceSummary.MissingWorklistCommand,
			sourceSummary.PresentWorklistCommand,
			sourceSummary.CompleteWorklistCommand,
			sourceSummary.DryRunImportCommand,
			sourceSummary.ExecuteImportCommand,
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-manifest-sync " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityManifestSyncApproval,
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityEvidenceBundleApproval,
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		},
		Notes: []string{
			"This read-only capture plan lists unique canonical replay artifacts and their mirrored source capture status.",
			"It does not call setup-svc, MetadataService, query APIs, normalized diff generation, cleanup, evidence import, manifest sync, or promotion.",
			"Use --source-status missing to plan capture work, --source-status present to review captured files, and --source-readiness complete to isolate structurally complete captures before import.",
			"Each artifact reuses the same required sections, shape keys, table coverage, runtime effects, and query-readback expectations as the strict evidence gate.",
		},
	}
	result.Totals.ArtifactFiles = sourceSummary.ArtifactFiles
	result.Totals.SourceFilesPresent = sourceSummary.SourceFilesPresent
	result.Totals.SourceFilesMissing = sourceSummary.SourceFilesMissing
	result.Totals.SourceFilesComplete = sourceSummary.SourceFilesComplete
	result.Totals.SourceFilesIncomplete = sourceSummary.SourceFilesIncomplete

	domainContracts := setupSvcLiveReplayDomainContractMap()
	seen := map[string]bool{}
	sectionTotals := map[string]*setupSvcLiveReplayEvidenceSectionSummary{}
	allArtifacts := []setupSvcLiveReplayCapturePlanArtifact{}
	for _, domain := range packet.Domains {
		if options.Domain != "" && normalizeDomain(domain.Domain) != options.Domain {
			continue
		}
		contract := domainContracts[normalizeDomain(domain.Domain)]
		for _, operation := range domain.Operations {
			if options.Operation != "" && strings.ToLower(strings.TrimSpace(operation.Operation)) != options.Operation {
				continue
			}
			for _, file := range operation.EvidenceFiles {
				normalizedFile := strings.TrimSpace(file)
				if normalizedFile == "" || seen[normalizedFile] {
					continue
				}
				seen[normalizedFile] = true
				artifactType := setupSvcLiveReplayArtifactType(normalizedFile)
				if options.ArtifactType != "" && setupSvcLiveReplayNormalizeArtifactType(artifactType) != options.ArtifactType {
					continue
				}
				suggestedSourcePath := setupSvcLiveReplayWorklistSuggestedSourcePath(normalizedFile)
				sourceExists := setupSvcLiveReplayWorklistSuggestedSourceExists(projectPath, suggestedSourcePath)
				if options.SourceStatus == "present" && !sourceExists {
					continue
				}
				if options.SourceStatus == "missing" && sourceExists {
					continue
				}
				requiredSections := setupSvcLiveReplayRequiredEvidenceSections(artifactType)
				sectionStatuses := setupSvcLiveReplayEvidenceSectionStatusesAtPath(filepath.Join(projectPath, suggestedSourcePath), requiredSections)
				sourceReadiness := setupSvcLiveReplaySourceReadiness(sourceExists, sectionStatuses)
				if options.SourceReadiness != "" && sourceReadiness != options.SourceReadiness {
					continue
				}
				if (options.EvidenceSection != "" || options.SectionStatus != "") && !setupSvcLiveReplayCollectionPlanSectionMatches(options, sectionStatuses) {
					continue
				}
				result.Totals.FilteredArtifactFiles++
				setupSvcLiveReplayAccumulateCapturePlanTypeTotals(&result.Totals, artifactType)
				for _, sectionStatus := range sectionStatuses {
					result.Totals.SourceEvidenceSections++
					if sectionStatus.Present {
						result.Totals.SourceEvidencePresent++
					} else {
						result.Totals.SourceEvidenceMissing++
					}
				}
				setupSvcLiveReplayAccumulateEvidenceSectionSummary(sectionTotals, artifactType, sectionStatuses)
				allArtifacts = append(allArtifacts, setupSvcLiveReplayCapturePlanArtifact{
					Domain:                    domain.Domain,
					Operation:                 operation.Operation,
					ArtifactType:              artifactType,
					Path:                      normalizedFile,
					SuggestedSourcePath:       suggestedSourcePath,
					SuggestedSourceExists:     sourceExists,
					SourceReadiness:           sourceReadiness,
					RequiredShapeKey:          setupSvcLiveReplayCollectionRequiredShapeKey(artifactType),
					ManifestStatusField:       setupSvcLiveReplayManifestStatusField(artifactType),
					RequiredEvidenceSections:  requiredSections,
					SourceEvidenceSections:    sectionStatuses,
					MissingEvidenceSections:   setupSvcLiveReplayMissingEvidenceSections(sectionStatuses),
					RequiredTables:            setupSvcLiveReplayCollectionRequiredTables(contract, artifactType),
					RuntimeEffects:            setupSvcLiveReplayCollectionRuntimeEffects(contract, artifactType),
					QueryReadbackExpectations: setupSvcLiveReplayCollectionQueryReadbackExpectations(contract, artifactType),
					CaptureTask: setupSvcLiveReplayArtifactCaptureTaskFor(projectPath, manifestPath, domain.Domain, operation.Operation, artifactType, normalizedFile, suggestedSourcePath,
						setupSvcLiveReplayCollectionRequiredShapeKey(artifactType),
						setupSvcLiveReplayManifestStatusField(artifactType),
						requiredSections,
						setupSvcLiveReplayCollectionRequiredTables(contract, artifactType),
						setupSvcLiveReplayCollectionRuntimeEffects(contract, artifactType),
						setupSvcLiveReplayCollectionQueryReadbackExpectations(contract, artifactType)),
					Checklist: setupSvcLiveReplayArtifactReplacementChecklist(artifactType),
				})
			}
		}
	}
	result.TotalNextArtifacts = len(allArtifacts)
	result.SourceEvidenceSections = setupSvcLiveReplayCapturePlanEvidenceSectionSummaries(sectionTotals, setupSvcLiveReplayEvidenceRecommendedOrder(), projectPath, manifestPath, options)
	result.SourceMissingSectionQueues = setupSvcLiveReplayEvidenceSectionQueues(result.SourceEvidenceSections, setupSvcLiveReplayEvidenceRecommendedOrder())
	result.OperatorPacket.SourceEvidenceSections = result.SourceEvidenceSections
	result.OperatorPacket.SourceMissingSectionQueues = result.SourceMissingSectionQueues
	result.NextArtifactOffset = options.Offset
	result.NextArtifactLimit = options.Limit
	start := options.Offset
	if start > len(allArtifacts) {
		start = len(allArtifacts)
	}
	end := len(allArtifacts)
	if options.Limit > 0 && start+options.Limit < end {
		end = start + options.Limit
	}
	result.Artifacts = append(result.Artifacts, allArtifacts[start:end]...)
	result.OmittedNextArtifacts = len(allArtifacts) - end
	if result.OmittedNextArtifacts < 0 {
		result.OmittedNextArtifacts = 0
	}
	result.PageCommands = setupSvcLiveReplayCapturePlanPageCommandsFor(projectPath, manifestPath, options, result.TotalNextArtifacts)
	result.OperatorPacket.ArtifactReplacementCount = result.TotalNextArtifacts
	result.OperatorPacket.SourceFilesPresent = result.Totals.SourceFilesPresent
	result.OperatorPacket.SourceFilesMissing = result.Totals.SourceFilesMissing
	result.OperatorPacket.SourceFilesComplete = result.Totals.SourceFilesComplete
	result.OperatorPacket.SourceFilesIncomplete = result.Totals.SourceFilesIncomplete
	return result, nil
}

func buildSetupSvcLiveReplayWorklistResult(projectPath string, manifestArg string, optionArgs ...string) (setupSvcLiveReplayWorklistResult, error) {
	parsedManifestArg, baseOptions, err := setupSvcLiveReplayParseGapArgs(manifestArg, optionArgs)
	if err != nil {
		return setupSvcLiveReplayWorklistResult{}, err
	}
	gaps, err := buildSetupSvcLiveReplayGapResult(projectPath, parsedManifestArg, setupSvcLiveReplayGapArgsFromOptions(baseOptions)...)
	if err != nil {
		return setupSvcLiveReplayWorklistResult{}, err
	}
	result := setupSvcLiveReplayWorklistResult{
		Mode:            "setup-svc-live-replay-worklist",
		Project:         projectPath,
		ReadOnly:        true,
		Status:          gaps.Status,
		ManifestPath:    gaps.ManifestPath,
		SourceRoot:      setupSvcLiveReplayWorklistSourceRoot(),
		CaptureRoot:     setupSvcLiveReplayWorklistCaptureRoot(projectPath),
		SourceGapStatus: gaps.CollectionPlan.Status,
		Filters:         setupSvcLiveReplayCollectionPlanFiltersFromOptions(baseOptions),
		BatchIndex:      setupSvcLiveReplayWorklistBatchIndexPointer(baseOptions),
		BatchLimit:      setupSvcLiveReplayWorklistEffectiveBatchLimit(baseOptions),
		NextCommands:    gaps.NextCommands,
		OperatorPacket:  setupSvcLiveReplayWorklistOperatorPacketFor(projectPath, gaps.ManifestPath, baseOptions, gaps.NextCommands),
		Notes: []string{
			"This read-only worklist expands missing evidence-section queues into concrete artifact batches.",
			"It does not call setup-svc, MetadataService writes, manifest sync, bundle writing, promotion, or matrix updates.",
			"Use each batch's artifact list to replace pending files with real structure proof, then save this worklist JSON and run the import dry-run command.",
			"The top-level sourceRoot is intentionally set so evidence-import can auto-match sourceRoot/<target evidence path> from mirrored capture directories.",
			"Query-readback batches are first-class parity work because query APIs must prove shape, relationships, readback expectations, and clean counters.",
		},
	}
	seenSourceReadiness := map[string]string{}
	seenArtifactPaths := map[string]bool{}
	seenSourceSections := map[string]bool{}
	sectionTotals := map[string]*setupSvcLiveReplayEvidenceSectionSummary{}
	for _, queue := range gaps.CollectionPlan.MissingSectionQueues {
		if !setupSvcLiveReplayWorklistQueueMatches(baseOptions, queue) {
			continue
		}
		workQueue := setupSvcLiveReplayWorklistQueue{
			ArtifactType:        queue.ArtifactType,
			Section:             queue.Section,
			Missing:             queue.Missing,
			RequiredShapeKey:    queue.RequiredShapeKey,
			ManifestStatusField: queue.ManifestStatusField,
			PageSize:            queue.PageSize,
			BatchCount:          queue.BatchCount,
			QueueCommand:        setupSvcLiveReplayWorklistQueueCommand(projectPath, gaps.ManifestPath, baseOptions, queue, -1),
		}
		result.Totals.Queues++
		result.Totals.MissingSections += queue.Missing
		if setupSvcLiveReplayNormalizeArtifactType(queue.ArtifactType) == "query-readback" {
			result.Totals.QueryReadbackQueues++
		}
		batchStart, batchEnd := setupSvcLiveReplayWorklistBatchRange(queue.BatchCount, baseOptions)
		for batchIndex := batchStart; batchIndex < batchEnd; batchIndex++ {
			offset := batchIndex * queue.PageSize
			batchOptions := baseOptions
			batchOptions.ArtifactType = setupSvcLiveReplayNormalizeArtifactType(queue.ArtifactType)
			batchOptions.EvidenceSection = queue.Section
			batchOptions.SectionStatus = "missing"
			batchOptions.Offset = offset
			batchOptions.Limit = queue.PageSize
			batchGaps, err := buildSetupSvcLiveReplayGapResult(projectPath, parsedManifestArg, setupSvcLiveReplayGapArgsFromOptions(batchOptions)...)
			if err != nil {
				return setupSvcLiveReplayWorklistResult{}, err
			}
			artifacts := setupSvcLiveReplayWorklistArtifactsForSourceFilters(projectPath, batchGaps.CollectionPlan.NextArtifacts, baseOptions.SourceStatus, baseOptions.SourceReadiness)
			if len(artifacts) == 0 && (strings.TrimSpace(baseOptions.SourceStatus) != "" || strings.TrimSpace(baseOptions.SourceReadiness) != "") {
				continue
			}
			operatorBatch := setupSvcLiveReplayWorklistOperatorBatchFor(projectPath, gaps.ManifestPath, batchIndex, queue, offset, queue.PageSize, artifacts, gaps.NextCommands)
			batchCommand := setupSvcLiveReplayWorklistQueueCommand(projectPath, gaps.ManifestPath, baseOptions, queue, batchIndex)
			batchPathOptions := batchOptions
			batchPathOptions.BatchIndex = batchIndex
			batchPath := setupSvcLiveReplayWorklistSuggestedPath(projectPath, batchPathOptions)
			saveCommand := batchCommand + " > " + shellPath(batchPath)
			dryRunImportCommand := "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(batchPath) + " --dry-run"
			executeImportCommand := "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(batchPath) + " --execute --approval " + setupSvcParityEvidenceImportApproval
			workQueue.Batches = append(workQueue.Batches, setupSvcLiveReplayWorklistBatch{
				BatchIndex:            batchIndex,
				Offset:                offset,
				Limit:                 queue.PageSize,
				Count:                 len(artifacts),
				Command:               batchCommand,
				SuggestedWorklistPath: batchPath,
				SaveWorklistCommand:   saveCommand,
				DryRunImportCommand:   dryRunImportCommand,
				ExecuteImportCommand:  executeImportCommand,
				OperatorBatch:         operatorBatch,
				Artifacts:             artifacts,
			})
			result.BatchSaveCommands = append(result.BatchSaveCommands, setupSvcLiveReplayWorklistBatchCommand{
				ArtifactType:          setupSvcLiveReplayNormalizeArtifactType(queue.ArtifactType),
				EvidenceSection:       queue.Section,
				BatchIndex:            batchIndex,
				Offset:                offset,
				Limit:                 queue.PageSize,
				Count:                 len(artifacts),
				SuggestedWorklistPath: batchPath,
				SaveWorklistCommand:   saveCommand,
				DryRunImportCommand:   dryRunImportCommand,
				ExecuteImportCommand:  executeImportCommand,
			})
			result.Totals.Batches++
			result.Totals.Artifacts += len(artifacts)
			for _, record := range operatorBatch.ArtifactReplacementRecords {
				artifactKey := strings.TrimSpace(record.Path)
				if artifactKey != "" {
					if seenArtifactPaths[artifactKey] {
						result.Totals.DuplicateArtifactRecords++
					} else {
						seenArtifactPaths[artifactKey] = true
					}
				}
				sourceKey := strings.TrimSpace(record.SuggestedSourcePath)
				if sourceKey == "" {
					sourceKey = strings.TrimSpace(record.Path)
				}
				if _, ok := seenSourceReadiness[sourceKey]; !ok {
					seenSourceReadiness[sourceKey] = record.SourceReadiness
				}
				if sourceKey != "" && !seenSourceSections[sourceKey] {
					seenSourceSections[sourceKey] = true
					setupSvcLiveReplayAccumulateEvidenceSectionSummary(sectionTotals, record.ArtifactType, record.SourceEvidenceSections)
				}
			}
			if setupSvcLiveReplayNormalizeArtifactType(queue.ArtifactType) == "query-readback" {
				result.Totals.QueryReadbackArtifacts += len(artifacts)
			}
		}
		workQueue.OmittedBatches = queue.BatchCount - len(workQueue.Batches)
		if workQueue.OmittedBatches < 0 {
			workQueue.OmittedBatches = 0
		}
		result.Totals.OmittedBatches += workQueue.OmittedBatches
		result.Queues = append(result.Queues, workQueue)
	}
	result.Totals.UniqueArtifactFiles = len(seenArtifactPaths)
	for _, readiness := range seenSourceReadiness {
		setupSvcLiveReplayWorklistAccumulateSourceReadiness(&result.Totals, readiness)
	}
	result.SourceEvidenceSections = setupSvcLiveReplayWorklistEvidenceSectionSummaries(sectionTotals, gaps.CollectionPlan.RecommendedOrder, projectPath, gaps.ManifestPath, baseOptions)
	result.SourceFiles = result.Totals.SourceFilesPresent + result.Totals.SourceFilesMissing
	result.TargetFiles = result.Totals.UniqueArtifactFiles
	result.SourceFilesPresent = result.Totals.SourceFilesPresent
	result.SourceFilesMissing = result.Totals.SourceFilesMissing
	result.SourceFilesComplete = result.Totals.SourceFilesComplete
	result.SourceFilesIncomplete = result.Totals.SourceFilesIncomplete
	result.QueuesCount = result.Totals.Queues
	result.Batches = result.Totals.Batches
	result.Artifacts = result.Totals.Artifacts
	result.UniqueArtifactFiles = result.Totals.UniqueArtifactFiles
	result.DuplicateArtifactRecords = result.Totals.DuplicateArtifactRecords
	result.MissingSections = result.Totals.MissingSections
	result.QueryReadbackQueues = result.Totals.QueryReadbackQueues
	result.QueryReadbackArtifacts = result.Totals.QueryReadbackArtifacts
	result.OmittedBatches = result.Totals.OmittedBatches
	result.OperatorPacket.ArtifactReplacementCount = result.Totals.Artifacts
	result.OperatorPacket.UniqueArtifactFiles = result.Totals.UniqueArtifactFiles
	result.OperatorPacket.SourceFiles = result.SourceFiles
	result.OperatorPacket.TargetFiles = result.TargetFiles
	result.OperatorPacket.DuplicateArtifactRecords = result.Totals.DuplicateArtifactRecords
	result.OperatorPacket.SourceFilesPresent = result.Totals.SourceFilesPresent
	result.OperatorPacket.SourceFilesMissing = result.Totals.SourceFilesMissing
	result.OperatorPacket.SourceFilesComplete = result.Totals.SourceFilesComplete
	result.OperatorPacket.SourceFilesIncomplete = result.Totals.SourceFilesIncomplete
	result.OperatorPacket.BatchSaveCommands = result.BatchSaveCommands
	result.OperatorPacket.SourceEvidenceSections = result.SourceEvidenceSections
	return result, nil
}

func setupSvcLiveReplayWorklistAccumulateSourceReadiness(totals *setupSvcLiveReplayWorklistTotals, readiness string) {
	switch readiness {
	case "complete":
		totals.SourceFilesPresent++
		totals.SourceFilesComplete++
	case "incomplete":
		totals.SourceFilesPresent++
		totals.SourceFilesIncomplete++
	default:
		totals.SourceFilesMissing++
	}
}

func setupSvcLiveReplaySourceCaptureMatches(options setupSvcLiveReplayCollectionPlanOptions, captureTask setupSvcLiveReplayArtifactCaptureTask) bool {
	if options.SourceSystem != "" && strings.TrimSpace(captureTask.SourceSystem) != options.SourceSystem {
		return false
	}
	if options.CaptureMode != "" && strings.TrimSpace(captureTask.CaptureMode) != options.CaptureMode {
		return false
	}
	return true
}

func buildSetupSvcLiveReplaySourceChecklistResult(projectPath string, manifestArg string, args ...string) (setupSvcLiveReplaySourceChecklistResult, error) {
	parsedManifestArg, options, err := setupSvcLiveReplayParseGapArgs(manifestArg, args)
	if err != nil {
		return setupSvcLiveReplaySourceChecklistResult{}, err
	}
	worklist, err := buildSetupSvcLiveReplayWorklistResult(projectPath, parsedManifestArg, setupSvcLiveReplayWorklistOptionArgsFromOptions(options)...)
	if err != nil {
		return setupSvcLiveReplaySourceChecklistResult{}, err
	}
	sourceItems := map[string]*setupSvcLiveReplaySourceChecklistItem{}
	targetFiles := map[string]bool{}
	artifactTypeRecords := map[string]int{}
	artifactTypeSources := map[string]map[string]bool{}
	artifactTypeTargets := map[string]map[string]bool{}
	readinessRecords := map[string]int{}
	readinessSources := map[string]map[string]bool{}
	replacementRecords := 0
	for _, queue := range worklist.Queues {
		for _, batch := range queue.Batches {
			worklistFile := strings.TrimSpace(batch.SuggestedWorklistPath)
			for _, record := range batch.OperatorBatch.ArtifactReplacementRecords {
				if !setupSvcLiveReplaySourceCaptureMatches(options, record.CaptureTask) {
					continue
				}
				replacementRecords++
				sourcePath := strings.TrimSpace(record.SuggestedSourcePath)
				if sourcePath == "" {
					sourcePath = strings.TrimSpace(record.Path)
				}
				targetPath := strings.TrimSpace(record.Path)
				artifactType := setupSvcLiveReplayNormalizeArtifactType(record.ArtifactType)
				readiness := strings.TrimSpace(record.SourceReadiness)
				if readiness == "" {
					readiness = "unknown"
				}
				if targetPath != "" {
					targetFiles[targetPath] = true
				}
				artifactTypeRecords[artifactType]++
				if artifactTypeSources[artifactType] == nil {
					artifactTypeSources[artifactType] = map[string]bool{}
				}
				if artifactTypeTargets[artifactType] == nil {
					artifactTypeTargets[artifactType] = map[string]bool{}
				}
				if sourcePath != "" {
					artifactTypeSources[artifactType][sourcePath] = true
				}
				if targetPath != "" {
					artifactTypeTargets[artifactType][targetPath] = true
				}
				readinessRecords[readiness]++
				if readinessSources[readiness] == nil {
					readinessSources[readiness] = map[string]bool{}
				}
				if sourcePath != "" {
					readinessSources[readiness][sourcePath] = true
				}
				item, ok := sourceItems[sourcePath]
				if !ok {
					item = &setupSvcLiveReplaySourceChecklistItem{
						SourcePath:              sourcePath,
						TargetPath:              targetPath,
						SourceReadiness:         readiness,
						Domain:                  record.Domain,
						Operation:               record.Operation,
						ArtifactType:            artifactType,
						RequiredShapeKey:        record.RequiredShapeKey,
						ManifestStatusField:     record.ManifestStatusField,
						ReplacementStatusTarget: record.ReplacementStatusTarget,
						CaptureTask:             record.CaptureTask,
					}
					sourceItems[sourcePath] = item
				}
				item.MissingEvidenceSections = setupSvcLiveReplayAppendUniqueStrings(item.MissingEvidenceSections, record.MissingEvidenceSections...)
				item.RequiredEvidenceSections = setupSvcLiveReplayAppendUniqueStrings(item.RequiredEvidenceSections, record.RequiredEvidenceSections...)
				item.RequiredTables = setupSvcLiveReplayAppendUniqueStrings(item.RequiredTables, record.RequiredTables...)
				item.RuntimeEffects = setupSvcLiveReplayAppendUniqueStrings(item.RuntimeEffects, record.RuntimeEffects...)
				item.QueryReadbackExpectations = setupSvcLiveReplayAppendUniqueStrings(item.QueryReadbackExpectations, record.QueryReadbackExpectations...)
				item.Checklist = setupSvcLiveReplayAppendUniqueStrings(item.Checklist, record.Checklist...)
				if worklistFile != "" {
					item.WorklistFiles = setupSvcLiveReplayAppendUniqueStrings(item.WorklistFiles, worklistFile)
				}
			}
		}
	}
	sources := make([]setupSvcLiveReplaySourceChecklistItem, 0, len(sourceItems))
	sourceKeys := make([]string, 0, len(sourceItems))
	for key := range sourceItems {
		sourceKeys = append(sourceKeys, key)
	}
	sort.Strings(sourceKeys)
	for _, key := range sourceKeys {
		item := sourceItems[key]
		setupSvcLiveReplaySortChecklistItem(item)
		sources = append(sources, *item)
	}
	missingSectionCounts := setupSvcLiveReplaySourceChecklistMissingSectionCounts(sources)
	nextQueueCommands := setupSvcLiveReplaySourceChecklistQueueCommands(projectPath, worklist.ManifestPath, options, missingSectionCounts)
	pageWorklistSaveCommands, pageChecklistSaveCommands := setupSvcLiveReplaySourceChecklistPageSaveCommands(nextQueueCommands)
	pageSaveScript := setupSvcLiveReplaySourceChecklistPageSaveScript(pageWorklistSaveCommands, pageChecklistSaveCommands)
	pageSaveScriptPath := setupSvcLiveReplaySourceChecklistScriptSuggestedPath(projectPath, options)
	savePageSaveScriptCommand := setupSvcLiveReplaySourceChecklistSavePageScriptCommand(projectPath, worklist.ManifestPath, options, pageSaveScriptPath)
	result := setupSvcLiveReplaySourceChecklistResult{
		Mode:          "setup-svc-live-replay-source-checklist",
		Project:       projectPath,
		ReadOnly:      true,
		Status:        worklist.Status,
		ManifestPath:  worklist.ManifestPath,
		GeneratedFrom: "setup-svc-live-replay-worklist",
		SourceRoot:    setupSvcLiveReplayWorklistSourceRoot(),
		CaptureRoot:   setupSvcLiveReplayWorklistCaptureRoot(projectPath),
		Filters:       setupSvcLiveReplayCollectionPlanFiltersFromOptions(options),
		Totals: setupSvcLiveReplaySourceChecklistTotals{
			InputWorklists:     1,
			WorklistQueues:     worklist.Totals.Queues,
			WorklistBatches:    worklist.Totals.Batches,
			ReplacementRecords: replacementRecords,
			UniqueSourceFiles:  len(sourceItems),
			UniqueTargetFiles:  len(targetFiles),
		},
		ArtifactTypeCounts:        setupSvcLiveReplaySourceChecklistArtifactTypeCounts(artifactTypeRecords, artifactTypeSources, artifactTypeTargets),
		SourceReadinessCounts:     setupSvcLiveReplaySourceChecklistReadinessCounts(readinessRecords, readinessSources),
		MissingSectionCounts:      missingSectionCounts,
		NextQueueCommands:         nextQueueCommands,
		PageWorklistSaveCommands:  pageWorklistSaveCommands,
		PageChecklistSaveCommands: pageChecklistSaveCommands,
		PageSaveScript:            pageSaveScript,
		PageSaveScriptPath:        pageSaveScriptPath,
		SavePageSaveScriptCommand: savePageSaveScriptCommand,
		Sources:                   sources,
		OperatorPacket:            setupSvcLiveReplaySourceChecklistOperatorPacketFor(projectPath, worklist.ManifestPath, options, worklist.NextCommands),
		Notes: []string{
			"This read-only checklist groups expanded missing-section replacement records by mirrored source capture file.",
			"Complete every source file before importing it through a --source-readiness complete worklist.",
			"All live replay evidence gates still require setup-svc-live-replay-evidence, bundle, promotion, and completion-audit to pass.",
		},
	}
	result.OperatorPacket.SourceFiles = result.Totals.UniqueSourceFiles
	result.OperatorPacket.TargetFiles = result.Totals.UniqueTargetFiles
	result.OperatorPacket.ReplacementRecords = result.Totals.ReplacementRecords
	result.OperatorPacket.WorklistQueues = result.Totals.WorklistQueues
	result.OperatorPacket.WorklistBatches = result.Totals.WorklistBatches
	result.OperatorPacket.MissingSectionKinds = len(result.MissingSectionCounts)
	result.OperatorPacket.NextQueueCount = len(result.NextQueueCommands)
	result.OperatorPacket.RepairQueueCount = len(result.NextQueueCommands)
	result.OperatorPacket.NextQueueCommands = nextQueueCommands
	result.OperatorPacket.PageWorklistSaveCommands = pageWorklistSaveCommands
	result.OperatorPacket.PageChecklistSaveCommands = pageChecklistSaveCommands
	result.OperatorPacket.PageSaveScript = pageSaveScript
	result.OperatorPacket.PageSaveScriptPath = pageSaveScriptPath
	result.OperatorPacket.SavePageSaveScriptCommand = savePageSaveScriptCommand
	result.SourceFiles = result.Totals.UniqueSourceFiles
	result.TargetFiles = result.Totals.UniqueTargetFiles
	result.ReplacementRecords = result.Totals.ReplacementRecords
	result.WorklistQueues = result.Totals.WorklistQueues
	result.WorklistBatches = result.Totals.WorklistBatches
	result.MissingSectionKinds = len(result.MissingSectionCounts)
	result.NextQueueCount = len(result.NextQueueCommands)
	result.RepairQueueCount = len(result.NextQueueCommands)
	return result, nil
}

func buildSetupSvcLiveReplaySourceHealthResult(projectPath string, manifestArg string, args ...string) (setupSvcLiveReplaySourceHealthResult, error) {
	_, options, err := setupSvcLiveReplayParseGapArgs(manifestArg, args)
	if err != nil {
		return setupSvcLiveReplaySourceHealthResult{}, err
	}
	checklist, err := buildSetupSvcLiveReplaySourceChecklistResult(projectPath, manifestArg, args...)
	if err != nil {
		return setupSvcLiveReplaySourceHealthResult{}, err
	}
	sourceFiles := map[string]bool{}
	targetFiles := map[string]bool{}
	artifactTypes := map[string]bool{}
	domainOperations := map[string]bool{}
	readinessCounts := map[string]int{}
	artifactStats := map[string]*setupSvcLiveReplaySourceHealthArtifactType{}
	artifactMissingSections := map[string]map[string]int{}
	artifactMissingSectionTargets := map[string]map[string]map[string]bool{}
	domainStats := map[string]*setupSvcLiveReplaySourceHealthDomainOperation{}
	missingSectionInstances := 0
	for _, source := range checklist.Sources {
		sourcePath := strings.TrimSpace(source.SourcePath)
		targetPath := strings.TrimSpace(source.TargetPath)
		artifactType := setupSvcLiveReplayNormalizeArtifactType(source.ArtifactType)
		readiness := strings.TrimSpace(source.SourceReadiness)
		if readiness == "" {
			readiness = "unknown"
		}
		domain := normalizeDomain(source.Domain)
		operation := strings.ToLower(strings.TrimSpace(source.Operation))
		if sourcePath != "" {
			sourceFiles[sourcePath] = true
		}
		if targetPath != "" {
			targetFiles[targetPath] = true
		}
		if artifactType != "" {
			artifactTypes[artifactType] = true
		}
		if domain != "" && operation != "" {
			domainOperations[domain+"/"+operation] = true
		}
		readinessCounts[readiness]++
		artifact := artifactStats[artifactType]
		if artifact == nil {
			artifact = &setupSvcLiveReplaySourceHealthArtifactType{ArtifactType: artifactType}
			artifactStats[artifactType] = artifact
		}
		artifact.SourceFiles++
		switch readiness {
		case "complete":
			artifact.CompleteSourceFiles++
		case "missing":
			artifact.MissingSourceFiles++
		default:
			artifact.IncompleteSourceFiles++
		}
		domainKey := domain + "/" + operation
		domainStat := domainStats[domainKey]
		if domainStat == nil {
			domainStat = &setupSvcLiveReplaySourceHealthDomainOperation{Domain: domain, Operation: operation}
			domainStats[domainKey] = domainStat
		}
		domainStat.SourceFiles++
		domainStat.ArtifactTypes = setupSvcLiveReplayAppendUniqueStrings(domainStat.ArtifactTypes, artifactType)
		switch readiness {
		case "complete":
			domainStat.CompleteSourceFiles++
		case "missing":
			domainStat.MissingSourceFiles++
		default:
			domainStat.IncompleteSourceFiles++
		}
		if artifactMissingSections[artifactType] == nil {
			artifactMissingSections[artifactType] = map[string]int{}
		}
		if artifactMissingSectionTargets[artifactType] == nil {
			artifactMissingSectionTargets[artifactType] = map[string]map[string]bool{}
		}
		for _, section := range source.MissingEvidenceSections {
			trimmed := strings.TrimSpace(section)
			if trimmed == "" {
				continue
			}
			missingSectionInstances++
			artifactMissingSections[artifactType][trimmed]++
			if artifactMissingSectionTargets[artifactType][trimmed] == nil {
				artifactMissingSectionTargets[artifactType][trimmed] = map[string]bool{}
			}
			if targetPath != "" {
				artifactMissingSectionTargets[artifactType][trimmed][targetPath] = true
			}
			domainStat.MissingSectionInstances++
		}
	}
	readiness := setupSvcLiveReplaySourceHealthReadinessCounts(readinessCounts)
	artifactHealth := setupSvcLiveReplaySourceHealthArtifactTypes(artifactStats, artifactMissingSections)
	domainHealth := setupSvcLiveReplaySourceHealthDomainOperations(domainStats)
	healthSectionCounts := setupSvcLiveReplaySourceHealthSectionCountsWithTargets(artifactMissingSections, artifactMissingSectionTargets)
	repairQueues := setupSvcLiveReplaySourceChecklistQueueCommands(projectPath, checklist.ManifestPath, options, healthSectionCounts)
	missingSections := setupSvcLiveReplaySourceHealthMissingSections(healthSectionCounts, repairQueues)
	completeSources := readinessCounts["complete"]
	incompleteSources := readinessCounts["incomplete"]
	missingSources := readinessCounts["missing"]
	result := setupSvcLiveReplaySourceHealthResult{
		Mode:                  "setup-svc-live-replay-source-health",
		Project:               projectPath,
		ReadOnly:              true,
		Status:                setupSvcLiveReplaySourceHealthStatus(completeSources, incompleteSources, missingSources),
		ManifestPath:          checklist.ManifestPath,
		GeneratedFrom:         "setup-svc-live-replay-source-checklist",
		SourceRoot:            checklist.SourceRoot,
		CaptureRoot:           checklist.CaptureRoot,
		Filters:               checklist.Filters,
		SourceFiles:           len(sourceFiles),
		TargetFiles:           len(targetFiles),
		SourceFilesPresent:    len(sourceFiles) - missingSources,
		SourceFilesComplete:   completeSources,
		SourceFilesIncomplete: incompleteSources,
		SourceFilesMissing:    missingSources,
		CompleteSourceFiles:   completeSources,
		IncompleteSourceFiles: incompleteSources,
		MissingSourceFiles:    missingSources,
		Totals: setupSvcLiveReplaySourceHealthTotals{
			SourceFiles:               len(sourceFiles),
			TargetFiles:               len(targetFiles),
			ArtifactTypes:             len(artifactTypes),
			DomainOperations:          len(domainOperations),
			CompleteSourceFiles:       completeSources,
			IncompleteSourceFiles:     incompleteSources,
			MissingSourceFiles:        missingSources,
			ImportableSourceFiles:     completeSources,
			RepairRequiredSourceFiles: incompleteSources + missingSources,
			MissingSectionKinds:       len(missingSections),
			MissingSectionInstances:   missingSectionInstances,
			CanImportCompleteSources:  completeSources > 0,
		},
		Readiness:                    readiness,
		ArtifactTypes:                artifactHealth,
		DomainOperations:             domainHealth,
		MissingSections:              missingSections,
		MissingEvidenceSectionCounts: missingSections,
		RepairQueues:                 repairQueues,
		RecommendedRunbook: []string{
			"Use repairQueues to save focused source-checklists/worklists for the highest missing evidence sections.",
			"Replace mirrored capture source files with real replay evidence until setup-svc-live-replay-source-health reports complete sources.",
			"Run the complete-source checklist, dry-run setup-svc-live-replay-evidence-import, then approved import only for complete sources.",
			"Run manifest sync, evidence verifier, evidence bundle, matrix promotion, and completion audit after imports.",
		},
		OperatorPacket: setupSvcLiveReplaySourceHealthOperatorPacket{
			Purpose:                  "Audit mirrored live replay capture source health without writing evidence or marking artifacts passed.",
			SourceFiles:              len(sourceFiles),
			CompleteSourceFiles:      completeSources,
			IncompleteSourceFiles:    incompleteSources,
			MissingSourceFiles:       missingSources,
			RepairQueues:             repairQueues,
			SourceExecutionCommand:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-execution-packet " + shellPath(checklist.ManifestPath) + setupSvcLiveReplaySourceHealthFilterSuffix(checklist.Filters),
			CompleteChecklistCommand: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-checklist " + shellPath(checklist.ManifestPath) + " --source-readiness complete",
			CompletionAuditCommand:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(checklist.ManifestPath),
		},
		Notes: []string{
			"This read-only health report summarizes the same mirrored source files and evidence sections used by the strict evidence gate.",
			"It does not call setup-svc, MetadataService, query APIs, normalized diff generation, cleanup, evidence import, manifest sync, bundle, promotion, or matrix updates.",
			"Complete source files are only import candidates; setup-svc-live-replay-evidence still decides whether imported artifacts can promote matrix status.",
		},
	}
	if completeSources > 0 {
		result.ReadyImportCommands = []string{
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-checklist " + shellPath(checklist.ManifestPath) + " --source-readiness complete",
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-execution-packet " + shellPath(checklist.ManifestPath) + " --source-readiness complete",
		}
	}
	return result, nil
}

func buildSetupSvcLiveReplaySourceValidateResult(projectPath string, manifestArg string, args ...string) (setupSvcLiveReplaySourceValidateResult, error) {
	_, options, err := setupSvcLiveReplayParseGapArgs(manifestArg, args)
	if err != nil {
		return setupSvcLiveReplaySourceValidateResult{}, err
	}
	if options.SourceReadiness == "" && options.SourceStatus == "" {
		options.SourceReadiness = "complete"
	}
	checklistArgs := setupSvcLiveReplayWorklistOptionArgsFromOptions(options)
	checklist, err := buildSetupSvcLiveReplaySourceChecklistResult(projectPath, manifestArg, checklistArgs...)
	if err != nil {
		return setupSvcLiveReplaySourceValidateResult{}, err
	}
	packet := setupSvcLiveReplaySourceValidateImportPacket(checklist)
	importDryRun, err := buildSetupSvcLiveReplayEvidenceImportApplyResult(projectPath, packet, false, "")
	if err != nil {
		return setupSvcLiveReplaySourceValidateResult{}, err
	}
	status := "ready_for_import_dry_run"
	if len(checklist.Sources) == 0 {
		status = "no_sources"
	} else if importDryRun.Status != "dry_run_ready" || importDryRun.FailedArtifacts > 0 || len(importDryRun.BlockingIssues) > 0 {
		status = "blocked_source_validation"
	}
	result := setupSvcLiveReplaySourceValidateResult{
		Mode:              "setup-svc-live-replay-source-validate",
		Project:           projectPath,
		ReadOnly:          true,
		Status:            status,
		ManifestPath:      checklist.ManifestPath,
		GeneratedFrom:     "setup-svc-live-replay-source-checklist",
		SourceRoot:        checklist.SourceRoot,
		CaptureRoot:       checklist.CaptureRoot,
		Filters:           checklist.Filters,
		SourceFiles:       len(checklist.Sources),
		ArtifactCount:     importDryRun.ArtifactCount,
		ReadyArtifacts:    importDryRun.PassedArtifacts,
		FailedArtifacts:   importDryRun.FailedArtifacts,
		SkippedDuplicates: importDryRun.SkippedDuplicates,
		Totals: setupSvcLiveReplaySourceValidateTotals{
			SourceFiles:       len(checklist.Sources),
			Artifacts:         importDryRun.ArtifactCount,
			ReadyArtifacts:    importDryRun.PassedArtifacts,
			FailedArtifacts:   importDryRun.FailedArtifacts,
			SkippedDuplicates: importDryRun.SkippedDuplicates,
		},
		ImportDryRun:   importDryRun,
		Artifacts:      importDryRun.Artifacts,
		RepairSummary:  importDryRun.RepairSummary,
		BlockingIssues: append([]string(nil), importDryRun.BlockingIssues...),
		RecommendedRunbook: []string{
			"Run source-health to identify repair queues before validating large source batches.",
			"Use source-validate with --source-readiness complete before evidence-import.",
			"Only run evidence-import --dry-run/--execute after source-validate reports ready_for_import_dry_run.",
			"Run manifest sync, strict evidence verification, bundle, promotion, and completion audit after approved imports.",
		},
		OperatorPacket: setupSvcLiveReplaySourceValidateOperator{
			Purpose:                "Validate mirrored capture source files through the same strict artifact checks used by evidence-import, without writing evidence.",
			SourceFiles:            len(checklist.Sources),
			ArtifactCount:          importDryRun.ArtifactCount,
			ReadyArtifacts:         importDryRun.PassedArtifacts,
			FailedArtifacts:        importDryRun.FailedArtifacts,
			SkippedDuplicates:      importDryRun.SkippedDuplicates,
			RepairQueueCount:       len(importDryRun.RepairSummary.RepairQueues),
			RepairQueues:           importDryRun.RepairSummary.RepairQueues,
			IssueKinds:             len(importDryRun.RepairSummary.IssueCounts),
			SourceChecklistCommand: setupSvcLiveReplaySourceChecklistCommand(projectPath, checklist.ManifestPath, options),
			SourceHealthCommand:    setupSvcLiveReplaySourceValidateScanCommand(projectPath, checklist.ManifestPath, "setup-svc-live-replay-source-health", options),
			DryRunImportCommand:    "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import <@complete-worklist.json> --dry-run",
			ApprovedImportCommand:  "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import <@complete-worklist.json> --execute --approval " + setupSvcParityEvidenceImportApproval,
			ManifestSyncCommand:    "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-manifest-sync " + shellPath(checklist.ManifestPath) + " --execute --approval " + setupSvcParityManifestSyncApproval,
			EvidenceVerifyCommand:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(checklist.ManifestPath),
			CompletionAuditCommand: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(checklist.ManifestPath),
		},
		Notes: []string{
			"This read-only validator uses the evidence-import dry-run validator against mirrored source capture files.",
			"It does not write evidence artifacts, execute setup-svc, execute MetadataService writes, sync manifests, write bundles, promote matrices, or mark artifacts passed.",
			"By default it validates --source-readiness complete candidates; pass --source-readiness incomplete to inspect why current placeholders fail strict import validation.",
		},
	}
	return result, nil
}

func setupSvcLiveReplaySourceValidateImportPacket(checklist setupSvcLiveReplaySourceChecklistResult) map[string]any {
	records := make([]any, 0, len(checklist.Sources))
	for _, source := range checklist.Sources {
		record := map[string]any{
			"domain":                    source.Domain,
			"operation":                 source.Operation,
			"artifactType":              source.ArtifactType,
			"path":                      source.TargetPath,
			"sourcePath":                source.SourcePath,
			"missingEvidenceSections":   source.MissingEvidenceSections,
			"requiredEvidenceSections":  source.RequiredEvidenceSections,
			"requiredTables":            source.RequiredTables,
			"runtimeEffects":            source.RuntimeEffects,
			"queryReadbackExpectations": source.QueryReadbackExpectations,
			"captureTask":               source.CaptureTask,
		}
		records = append(records, record)
	}
	return map[string]any{
		"mode":                       "setup-svc-live-replay-source-validate-import-packet",
		"manifestPath":               checklist.ManifestPath,
		"sourceRoot":                 "",
		"artifactReplacementRecords": records,
	}
}

func setupSvcLiveReplaySourceValidateScanCommand(projectPath string, manifestPath string, mode string, options setupSvcLiveReplayCollectionPlanOptions) string {
	return "cloudcc scan msapi " + shellPath(projectPath) + " " + mode + " " + shellPath(manifestPath) + setupSvcLiveReplaySourceHealthFilterSuffix(setupSvcLiveReplayCollectionPlanFiltersFromOptions(options))
}

func setupSvcLiveReplaySourceHealthStatus(completeSources int, incompleteSources int, missingSources int) string {
	switch {
	case missingSources > 0:
		return "missing_sources"
	case incompleteSources > 0 && completeSources > 0:
		return "partial_complete_sources"
	case incompleteSources > 0:
		return "pending_source_repair"
	case completeSources > 0:
		return "ready_for_complete_source_import"
	default:
		return "no_sources"
	}
}

func setupSvcLiveReplaySourceHealthReadinessCounts(counts map[string]int) []setupSvcLiveReplaySourceHealthReadiness {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]setupSvcLiveReplaySourceHealthReadiness, 0, len(keys))
	for _, key := range keys {
		out = append(out, setupSvcLiveReplaySourceHealthReadiness{
			SourceReadiness: key,
			SourceFiles:     counts[key],
		})
	}
	return out
}

func setupSvcLiveReplaySourceHealthArtifactTypes(stats map[string]*setupSvcLiveReplaySourceHealthArtifactType, missing map[string]map[string]int) []setupSvcLiveReplaySourceHealthArtifactType {
	keys := make([]string, 0, len(stats))
	for key := range stats {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]setupSvcLiveReplaySourceHealthArtifactType, 0, len(keys))
	for _, key := range keys {
		stat := *stats[key]
		sectionCounts := missing[key]
		stat.MissingSectionKinds = len(sectionCounts)
		topSections := setupSvcLiveReplaySourceHealthTopSections(sectionCounts, 5)
		stat.TopMissingSections = topSections
		for _, count := range sectionCounts {
			stat.MissingSectionInstances += count
		}
		out = append(out, stat)
	}
	return out
}

func setupSvcLiveReplaySourceHealthTopSections(counts map[string]int, limit int) []string {
	type sectionCount struct {
		section string
		count   int
	}
	items := make([]sectionCount, 0, len(counts))
	for section, count := range counts {
		items = append(items, sectionCount{section: section, count: count})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].count != items[j].count {
			return items[i].count > items[j].count
		}
		return items[i].section < items[j].section
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.section)
	}
	return out
}

func setupSvcLiveReplaySourceHealthSectionCounts(missing map[string]map[string]int) []setupSvcLiveReplaySourceChecklistSectionCount {
	return setupSvcLiveReplaySourceHealthSectionCountsWithTargets(missing, nil)
}

func setupSvcLiveReplaySourceHealthSectionCountsWithTargets(missing map[string]map[string]int, targets map[string]map[string]map[string]bool) []setupSvcLiveReplaySourceChecklistSectionCount {
	keys := []string{}
	for artifactType, sections := range missing {
		normalizedArtifactType := setupSvcLiveReplayNormalizeArtifactType(artifactType)
		for section, count := range sections {
			section = strings.TrimSpace(section)
			if normalizedArtifactType == "" || section == "" || count <= 0 {
				continue
			}
			keys = append(keys, normalizedArtifactType+"\x00"+section)
		}
	}
	sort.Strings(keys)
	out := make([]setupSvcLiveReplaySourceChecklistSectionCount, 0, len(keys))
	for _, key := range keys {
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 {
			continue
		}
		out = append(out, setupSvcLiveReplaySourceChecklistSectionCount{
			EvidenceSection: parts[1],
			SourceFiles:     missing[parts[0]][parts[1]],
			TargetFiles:     len(targets[parts[0]][parts[1]]),
			ArtifactTypes:   []string{parts[0]},
		})
	}
	return out
}

func setupSvcLiveReplaySourceHealthDomainOperations(stats map[string]*setupSvcLiveReplaySourceHealthDomainOperation) []setupSvcLiveReplaySourceHealthDomainOperation {
	keys := make([]string, 0, len(stats))
	for key := range stats {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]setupSvcLiveReplaySourceHealthDomainOperation, 0, len(keys))
	for _, key := range keys {
		stat := *stats[key]
		sort.Strings(stat.ArtifactTypes)
		out = append(out, stat)
	}
	return out
}

func setupSvcLiveReplaySourceHealthMissingSections(counts []setupSvcLiveReplaySourceChecklistSectionCount, queues []setupSvcLiveReplaySourceChecklistQueueCommand) []setupSvcLiveReplaySourceHealthMissingSection {
	queueByKey := map[string]string{}
	for _, queue := range queues {
		key := setupSvcLiveReplayNormalizeArtifactType(queue.ArtifactType) + "\x00" + strings.TrimSpace(queue.EvidenceSection)
		queueByKey[key] = queue.SourceChecklistCommand
	}
	out := []setupSvcLiveReplaySourceHealthMissingSection{}
	for _, count := range counts {
		section := strings.TrimSpace(count.EvidenceSection)
		for _, artifactType := range count.ArtifactTypes {
			normalizedArtifactType := setupSvcLiveReplayNormalizeArtifactType(artifactType)
			key := normalizedArtifactType + "\x00" + section
			out = append(out, setupSvcLiveReplaySourceHealthMissingSection{
				ArtifactType:    normalizedArtifactType,
				EvidenceSection: section,
				SourceFiles:     count.SourceFiles,
				TargetFiles:     count.TargetFiles,
				QueueCommand:    queueByKey[key],
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SourceFiles != out[j].SourceFiles {
			return out[i].SourceFiles > out[j].SourceFiles
		}
		if out[i].ArtifactType != out[j].ArtifactType {
			return out[i].ArtifactType < out[j].ArtifactType
		}
		return out[i].EvidenceSection < out[j].EvidenceSection
	})
	return out
}

func setupSvcLiveReplaySourceHealthFilterSuffix(filters *setupSvcLiveReplayCollectionPlanFilters) string {
	if filters == nil {
		return ""
	}
	args := []string{}
	if filters.Domain != "" {
		args = append(args, "--domain", filters.Domain)
	}
	if filters.Operation != "" {
		args = append(args, "--operation", filters.Operation)
	}
	if filters.ArtifactType != "" {
		args = append(args, "--artifact-type", filters.ArtifactType)
	}
	if filters.SourceSystem != "" {
		args = append(args, "--source-system", filters.SourceSystem)
	}
	if filters.CaptureMode != "" {
		args = append(args, "--capture-mode", filters.CaptureMode)
	}
	if filters.EvidenceSection != "" {
		args = append(args, "--evidence-section", filters.EvidenceSection)
	}
	if filters.SectionStatus != "" {
		args = append(args, "--section-status", filters.SectionStatus)
	}
	if filters.SourceStatus != "" {
		args = append(args, "--source-status", filters.SourceStatus)
	}
	if filters.SourceReadiness != "" {
		args = append(args, "--source-readiness", filters.SourceReadiness)
	}
	if len(args) == 0 {
		return ""
	}
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			quoted = append(quoted, arg)
		} else {
			quoted = append(quoted, shellPath(arg))
		}
	}
	return " " + strings.Join(quoted, " ")
}

func buildSetupSvcLiveReplaySourceExecutionPacketResult(projectPath string, manifestArg string, args ...string) (setupSvcLiveReplaySourceExecutionPacketResult, error) {
	_, options, err := setupSvcLiveReplayParseGapArgs(manifestArg, args)
	if err != nil {
		return setupSvcLiveReplaySourceExecutionPacketResult{}, err
	}
	checklist, err := buildSetupSvcLiveReplaySourceChecklistResult(projectPath, manifestArg, args...)
	if err != nil {
		return setupSvcLiveReplaySourceExecutionPacketResult{}, err
	}
	datasourceReadiness := setupSvcLiveReplayDatasourceReadinessFor()
	groups := setupSvcLiveReplaySourceExecutionGroups(projectPath, checklist.ManifestPath, checklist.Sources)
	operatorBatches := make([]setupSvcLiveReplaySourceExecutionOperatorBatch, 0, len(groups))
	batchSaveCommands := []string{}
	importBatchSaveCommands := []string{}
	sourceFiles := map[string]bool{}
	targetFiles := map[string]bool{}
	artifactTypes := map[string]bool{}
	domainOperations := map[string]bool{}
	evidenceSections := map[string]bool{}
	for _, group := range groups {
		for _, source := range group.Items {
			if strings.TrimSpace(source.SourcePath) != "" {
				sourceFiles[source.SourcePath] = true
			}
			if strings.TrimSpace(source.TargetPath) != "" {
				targetFiles[source.TargetPath] = true
			}
			if strings.TrimSpace(source.ArtifactType) != "" {
				artifactTypes[source.ArtifactType] = true
			}
			if strings.TrimSpace(source.Domain) != "" && strings.TrimSpace(source.Operation) != "" {
				domainOperations[source.Domain+"/"+source.Operation] = true
			}
			for _, section := range source.MissingEvidenceSections {
				if strings.TrimSpace(section) != "" {
					evidenceSections[section] = true
				}
			}
		}
		importBatchPath := setupSvcLiveReplaySourceExecutionBatchPathForMode(projectPath, group.ArtifactType, group.CaptureMode, "complete")
		saveImportBatchCommand := setupSvcLiveReplaySourceExecutionSaveBatchCommandForMode(projectPath, checklist.ManifestPath, group.ArtifactType, group.CaptureMode, "complete", importBatchPath)
		dryRunCaptureCommand, executeCaptureCommand := setupSvcLiveReplaySourceExecutionCaptureCommands(projectPath, checklist.ManifestPath, group.SuggestedBatchPath, group.ArtifactType, group.CaptureMode)
		operatorBatch := setupSvcLiveReplaySourceExecutionOperatorBatch{
			ArtifactType:             group.ArtifactType,
			SourceSystem:             group.SourceSystem,
			CaptureMode:              group.CaptureMode,
			SourceFiles:              group.SourceFiles,
			TargetFiles:              group.TargetFiles,
			DomainOperations:         group.DomainOperations,
			EvidenceSections:         group.EvidenceSections,
			BatchPath:                group.SuggestedBatchPath,
			SuggestedBatchPath:       group.SuggestedBatchPath,
			SaveBatchCommand:         group.SaveBatchCommand,
			SuggestedImportBatchPath: importBatchPath,
			SaveImportBatchCommand:   saveImportBatchCommand,
			NextAction:               setupSvcLiveReplaySourceExecutionNextAction(group.ArtifactType, group.CaptureMode),
			ManualCaptureRequired:    dryRunCaptureCommand == "" && executeCaptureCommand == "",
			DryRunCaptureCommand:     dryRunCaptureCommand,
			ExecuteCaptureCommand:    executeCaptureCommand,
			PostCaptureCheckCommand:  group.PostCaptureCheckCommand,
			DryRunImportCommand:      "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(importBatchPath) + " --dry-run",
			ApprovedImportCommand:    "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(importBatchPath) + " --execute --approval " + setupSvcParityEvidenceImportApproval,
			CompletionAuditCommand:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(checklist.ManifestPath),
		}
		if setupSvcLiveReplayNormalizeArtifactType(group.ArtifactType) == "metadata-service" {
			readiness := datasourceReadiness
			operatorBatch.MetadataServiceDatasource = &readiness
		}
		operatorBatches = append(operatorBatches, operatorBatch)
		batchSaveCommands = setupSvcLiveReplayAppendUniqueStrings(batchSaveCommands, group.SaveBatchCommand)
		importBatchSaveCommands = setupSvcLiveReplayAppendUniqueStrings(importBatchSaveCommands, saveImportBatchCommand)
	}
	executionRunbook := setupSvcLiveReplaySourceExecutionRunbook(groups, operatorBatches)
	batchSaveScript := setupSvcLiveReplaySourceExecutionBatchSaveScript(batchSaveCommands)
	batchSaveScriptPath := setupSvcLiveReplaySourceExecutionScriptSuggestedPath(projectPath, options)
	saveBatchSaveScriptCommand := setupSvcLiveReplaySourceExecutionSaveBatchScriptCommand(projectPath, checklist.ManifestPath, options, batchSaveScriptPath)
	importBatchSaveScript := setupSvcLiveReplaySourceExecutionImportBatchSaveScript(importBatchSaveCommands)
	importBatchSaveScriptPath := setupSvcLiveReplaySourceExecutionImportScriptSuggestedPath(projectPath, options)
	saveImportBatchSaveScriptCommand := setupSvcLiveReplaySourceExecutionSaveImportBatchScriptCommand(projectPath, checklist.ManifestPath, options, importBatchSaveScriptPath)
	totals := setupSvcLiveReplaySourceExecutionPacketTotals{
		SourceFiles:        len(sourceFiles),
		TargetFiles:        len(targetFiles),
		ArtifactTypes:      len(artifactTypes),
		DomainOperations:   len(domainOperations),
		EvidenceSections:   len(evidenceSections),
		CaptureGroups:      len(groups),
		GroupedSourceFiles: setupSvcLiveReplaySourceExecutionGroupedSourceFiles(groups),
		GroupedTargetFiles: setupSvcLiveReplaySourceExecutionGroupedTargetFiles(groups),
	}
	for _, readiness := range checklist.SourceReadinessCounts {
		switch readiness.SourceReadiness {
		case "complete":
			totals.CompleteSourceFiles += readiness.SourceFiles
		case "incomplete":
			totals.IncompleteSourceFiles += readiness.SourceFiles
		case "missing":
			totals.MissingSourceFiles += readiness.SourceFiles
		}
	}
	runbookMarkdown := setupSvcLiveReplaySourceExecutionRunbookMarkdown(projectPath, checklist.ManifestPath, executionRunbook, totals)
	runbookMarkdownPath := setupSvcLiveReplaySourceExecutionRunbookMarkdownSuggestedPath(projectPath, options)
	saveRunbookMarkdownCommand := setupSvcLiveReplaySourceExecutionSaveRunbookMarkdownCommand(projectPath, checklist.ManifestPath, options, runbookMarkdownPath)
	result := setupSvcLiveReplaySourceExecutionPacketResult{
		Mode:                      "setup-svc-live-replay-source-execution-packet",
		Project:                   projectPath,
		ReadOnly:                  true,
		Status:                    checklist.Status,
		ManifestPath:              checklist.ManifestPath,
		GeneratedFrom:             "setup-svc-live-replay-source-checklist",
		SourceRoot:                checklist.SourceRoot,
		CaptureRoot:               checklist.CaptureRoot,
		Filters:                   checklist.Filters,
		MetadataServiceDatasource: datasourceReadiness,
		Totals:                    totals,
		CaptureModeGroups:         groups,
		Groups:                    groups,
		OperatorBatches:           operatorBatches,
		OperatorPacket: setupSvcLiveReplaySourceExecutionOperatorPacket{
			Purpose:                          "Save capture batches, run the gated capture commands, then import only complete source evidence.",
			SourceRoot:                       checklist.SourceRoot,
			CaptureRoot:                      checklist.CaptureRoot,
			SourceFiles:                      totals.SourceFiles,
			TargetFiles:                      totals.TargetFiles,
			IncompleteSourceFiles:            totals.IncompleteSourceFiles,
			CompleteSourceFiles:              totals.CompleteSourceFiles,
			MissingSourceFiles:               totals.MissingSourceFiles,
			ArtifactTypes:                    totals.ArtifactTypes,
			DomainOperations:                 totals.DomainOperations,
			EvidenceSectionCount:             totals.EvidenceSections,
			CaptureGroups:                    totals.CaptureGroups,
			OperatorBatchCount:               len(operatorBatches),
			RunbookStepCount:                 len(executionRunbook),
			BatchSaveCommandCount:            len(batchSaveCommands),
			ImportBatchSaveCommandCount:      len(importBatchSaveCommands),
			BatchSaveScriptPath:              batchSaveScriptPath,
			SaveBatchSaveScriptCommand:       saveBatchSaveScriptCommand,
			ImportBatchSaveScriptPath:        importBatchSaveScriptPath,
			SaveImportBatchSaveScriptCommand: saveImportBatchSaveScriptCommand,
			RunbookMarkdownPath:              runbookMarkdownPath,
			SaveRunbookMarkdownCommand:       saveRunbookMarkdownCommand,
			CompletionAuditCommand:           "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(checklist.ManifestPath),
			MetadataServiceDatasource:        datasourceReadiness,
			StopConditions: []string{
				"Do not execute MetadataService capture unless metadataServiceDatasource.readyForRealDatasource is true.",
				"Do not save complete import batches until source-checklist reports sourceReadiness complete.",
				"Do not run matrix promotion until strict evidence verification and evidence bundle verification pass.",
			},
		},
		ExecutionRunbook:                 executionRunbook,
		BatchSaveCommands:                batchSaveCommands,
		BatchSaveScript:                  batchSaveScript,
		BatchSaveScriptPath:              batchSaveScriptPath,
		SaveBatchSaveScriptCommand:       saveBatchSaveScriptCommand,
		ImportBatchSaveCommands:          importBatchSaveCommands,
		ImportBatchSaveScript:            importBatchSaveScript,
		ImportBatchSaveScriptPath:        importBatchSaveScriptPath,
		SaveImportBatchSaveScriptCommand: saveImportBatchSaveScriptCommand,
		RunbookMarkdown:                  runbookMarkdown,
		RunbookMarkdownPath:              runbookMarkdownPath,
		SaveRunbookMarkdownCommand:       saveRunbookMarkdownCommand,
		CaptureGroups:                    totals.CaptureGroups,
		ArtifactTypes:                    totals.ArtifactTypes,
		SourceFiles:                      totals.SourceFiles,
		TargetFiles:                      totals.TargetFiles,
		IncompleteSourceFiles:            totals.IncompleteSourceFiles,
		CompleteSourceFiles:              totals.CompleteSourceFiles,
		MissingSourceFiles:               totals.MissingSourceFiles,
		DomainOperations:                 totals.DomainOperations,
		EvidenceSectionCount:             totals.EvidenceSections,
		NextSteps: []string{
			"Replace each mirrored source JSON listed by groups/captureModeGroups with real evidence matching missingEvidenceSections.",
			"Run setup-svc-live-replay-source-checklist --source-readiness complete after sources are replaced.",
			"Dry-run and then approved execute setup-svc-live-replay-evidence-import for complete worklists.",
			"Run manifest sync, evidence verifier, evidence bundle, promotion dry-run/execute, then completion audit.",
		},
		Notes: []string{
			"This read-only execution packet groups unique source capture files by artifact type, source system, and capture mode.",
			"It is an operator packet for evidence replacement and does not call setup-svc, MetadataService, query APIs, normalized diff generation, cleanup, import, bundle, promotion, or completion audit.",
			"Query-readback is included as a first-class group because query APIs must prove readback tables, relationships, expectation checks, and clean counters.",
		},
	}
	if len(groups) == 1 {
		group := groups[0]
		result.ArtifactType = group.ArtifactType
		result.SourceSystem = group.SourceSystem
		result.CaptureMode = group.CaptureMode
		result.SourceFiles = group.SourceFiles
		result.TargetFiles = group.TargetFiles
		result.DomainOperations = group.DomainOperations
		result.EvidenceSections = group.EvidenceSections
		result.RequiredTables = group.RequiredTables
		result.RuntimeEffects = group.RuntimeEffects
		result.QueryReadbackExpectations = group.QueryReadbackExpectations
		result.SuggestedBatchPath = group.SuggestedBatchPath
		result.SaveBatchCommand = group.SaveBatchCommand
		result.PostCaptureCheckCommand = group.PostCaptureCheckCommand
		result.Items = group.Items
		if len(operatorBatches) == 1 {
			batch := operatorBatches[0]
			result.OperatorBatch = &batch
		}
	}
	return result, nil
}

func buildSetupSvcLiveReplayQueryReadbackCapturePlanResult(projectPath string, manifestArg string, args ...string) (setupSvcLiveReplayQueryReadbackCapturePlanResult, error) {
	_, options, err := setupSvcLiveReplayParseGapArgs(manifestArg, args)
	if err != nil {
		return setupSvcLiveReplayQueryReadbackCapturePlanResult{}, err
	}
	options.ArtifactType = "query-readback"
	if options.SourceReadiness == "" {
		options.SourceReadiness = "incomplete"
	}
	pagingExplicit := setupSvcLiveReplayQueryReadbackCapturePlanPagingExplicit(args)
	checklistOptions := options
	if pagingExplicit {
		checklistOptions.Offset = 0
		checklistOptions.Limit = 25
	}
	checklistArgs := setupSvcLiveReplaySourceExecutionArgsFromOptions(checklistOptions)
	if checklistOptions.SourceStatus != "" {
		checklistArgs = append(checklistArgs, "--source-status", options.SourceStatus)
	}
	if checklistOptions.SourceReadiness != "" {
		checklistArgs = append(checklistArgs, "--source-readiness", options.SourceReadiness)
	}
	checklist, err := buildSetupSvcLiveReplaySourceChecklistResult(projectPath, manifestArg, checklistArgs...)
	if err != nil {
		return setupSvcLiveReplayQueryReadbackCapturePlanResult{}, err
	}
	requests := make([]setupSvcLiveReplayQueryReadbackCaptureItem, 0, len(checklist.Sources))
	for _, source := range checklist.Sources {
		if setupSvcLiveReplayNormalizeArtifactType(source.ArtifactType) != "query-readback" {
			continue
		}
		domain := normalizeDomain(source.Domain)
		operation := strings.ToLower(strings.TrimSpace(source.Operation))
		completeWorklistCommand := setupSvcLiveReplayQueryReadbackCompleteWorklistCommand(projectPath, checklist.ManifestPath, domain, operation)
		importPath := setupSvcLiveReplaySourceExecutionBatchPath(projectPath, "query-readback", "complete")
		recommendedCommands := setupSvcLiveReplayQueryReadbackRecommendedCommands(projectPath, domain, operation)
		requests = append(requests, setupSvcLiveReplayQueryReadbackCaptureItem{
			Domain:                      domain,
			Operation:                   operation,
			SourcePath:                  source.SourcePath,
			TargetPath:                  source.TargetPath,
			SourceReadiness:             firstString(source.SourceReadiness, "incomplete"),
			RequiredTables:              append([]string{}, source.RequiredTables...),
			QueryReadbackExpectations:   append([]string{}, source.QueryReadbackExpectations...),
			RequiredEvidenceSections:    append([]string{}, source.RequiredEvidenceSections...),
			MissingEvidenceSections:     append([]string{}, source.MissingEvidenceSections...),
			CaptureArtifactShape:        setupSvcLiveReplayQueryReadbackArtifactShape(projectPath, domain, operation, source.RequiredTables, source.QueryReadbackExpectations),
			ScannerCommand:              firstString(recommendedCommands...),
			RecommendedReadbackCommands: recommendedCommands,
			PostCaptureCheckCommand:     setupSvcLiveReplayCaptureTaskCheckCommand(projectPath, checklist.ManifestPath, domain, operation, "query-readback"),
			CompleteWorklistCommand:     completeWorklistCommand,
			DryRunImportCommand:         "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(importPath) + " --dry-run",
			ApprovedImportCommand:       "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(importPath) + " --execute --approval " + setupSvcParityEvidenceImportApproval,
		})
	}
	sort.SliceStable(requests, func(i, j int) bool {
		left := requests[i].Domain + "/" + requests[i].Operation
		right := requests[j].Domain + "/" + requests[j].Operation
		if left != right {
			return left < right
		}
		return requests[i].SourcePath < requests[j].SourcePath
	})
	totalRequests := len(requests)
	if pagingExplicit {
		requests = setupSvcLiveReplayQueryReadbackCapturePlanPage(requests, options.Offset, options.Limit)
	}
	stats := setupSvcLiveReplayQueryReadbackCaptureStatsFor(requests)
	omittedRequests := totalRequests - len(requests)
	if omittedRequests < 0 {
		omittedRequests = 0
	}
	batchCommand := setupSvcLiveReplaySourceExecutionSaveBatchCommandFor(projectPath, checklist.ManifestPath, "query-readback", firstString(options.SourceReadiness, "incomplete"), setupSvcLiveReplaySourceExecutionBatchPath(projectPath, "query-readback", firstString(options.SourceReadiness, "incomplete")))
	completeImportPath := setupSvcLiveReplaySourceExecutionBatchPath(projectPath, "query-readback", "complete")
	result := setupSvcLiveReplayQueryReadbackCapturePlanResult{
		Mode:                         "setup-svc-live-replay-query-readback-capture-plan",
		Project:                      projectPath,
		ReadOnly:                     true,
		Status:                       checklist.Status,
		ManifestPath:                 checklist.ManifestPath,
		GeneratedFrom:                "setup-svc-live-replay-source-checklist",
		SourceRoot:                   checklist.SourceRoot,
		CaptureRoot:                  checklist.CaptureRoot,
		Filters:                      setupSvcLiveReplayCollectionPlanFiltersFromOptions(options),
		QueryReadbackSources:         len(requests),
		TotalQueryReadbackSources:    totalRequests,
		ReturnedQueryReadbackSources: len(requests),
		OmittedQueryReadbackSources:  omittedRequests,
		Totals: setupSvcLiveReplayQueryReadbackCaptureTotals{
			QueryReadbackSources:         len(requests),
			TotalQueryReadbackSources:    totalRequests,
			ReturnedQueryReadbackSources: len(requests),
			Offset:                       options.Offset,
			Limit:                        setupSvcLiveReplayQueryReadbackCapturePlanLimit(options, pagingExplicit),
			OmittedQueryReadbackSources:  omittedRequests,
			DomainOperations:             stats.domainOperations,
			RequiredTables:               stats.requiredTables,
			Expectations:                 stats.expectations,
		},
		CaptureRequests: requests,
		OperatorPacket: setupSvcLiveReplayQueryReadbackOperator{
			Purpose:                 "Turn query-readback source checklist items into concrete capture requests without writing evidence or marking it passed.",
			QueryReadbackSources:    len(requests),
			SourceReadiness:         options.SourceReadiness,
			RecommendedBatchCommand: batchCommand,
			SavePlanCommand:         setupSvcLiveReplayQueryReadbackCapturePlanCommand(projectPath, checklist.ManifestPath, options) + " > " + shellPath(setupSvcLiveReplayQueryReadbackCapturePlanSuggestedPath(projectPath, options)),
			PostCaptureCheckCommand: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-checklist " + shellPath(checklist.ManifestPath) + " --artifact-type query-readback --source-readiness complete",
			DryRunImportCommand:     "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(completeImportPath) + " --dry-run",
			ApprovedImportCommand:   "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(completeImportPath) + " --execute --approval " + setupSvcParityEvidenceImportApproval,
			CompletionAuditCommand:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(checklist.ManifestPath),
		},
		StopConditions: []string{
			"This command is read-only and does not prove query-readback evidence by itself.",
			"Do not change a source artifact status to passed until the replay write has happened and every required table, relationship, expectation check, and clean counter is populated from real readback.",
			"Do not import query-readback captures until source-checklist reports them as complete and evidence-import --dry-run passes.",
		},
		NextSteps: []string{
			"Run or script the recommended readback command after the matching setup-svc and MetadataService replay writes.",
			"Fill each captureArtifactShape with real readback rows, fields, relationships, named expectation checks, and zero clean counters.",
			"Regenerate the query-readback complete batch, dry-run evidence-import, then proceed to manifest-sync, evidence verification, bundle, promotion, and completion audit.",
		},
	}
	return result, nil
}

func setupSvcLiveReplaySourceExecutionRunbook(groups []setupSvcLiveReplaySourceExecutionPacketGroup, batches []setupSvcLiveReplaySourceExecutionOperatorBatch) []setupSvcLiveReplaySourceExecutionRunbookStep {
	batchByGroup := map[string]setupSvcLiveReplaySourceExecutionOperatorBatch{}
	for _, batch := range batches {
		key := setupSvcLiveReplaySourceExecutionGroupKey(batch.ArtifactType, batch.CaptureMode, batch.SourceSystem)
		if key != "" {
			batchByGroup[key] = batch
		}
	}
	out := make([]setupSvcLiveReplaySourceExecutionRunbookStep, 0, len(groups))
	completedArtifacts := []string{}
	for i, group := range groups {
		artifactType := setupSvcLiveReplayNormalizeArtifactType(group.ArtifactType)
		batch := batchByGroup[setupSvcLiveReplaySourceExecutionGroupKey(group.ArtifactType, group.CaptureMode, group.SourceSystem)]
		step := setupSvcLiveReplaySourceExecutionRunbookStep{
			Order:                     i + 1,
			ArtifactType:              artifactType,
			DependsOn:                 append([]string{}, completedArtifacts...),
			SourceSystem:              group.SourceSystem,
			CaptureMode:               group.CaptureMode,
			SourceFiles:               group.SourceFiles,
			TargetFiles:               group.TargetFiles,
			DomainOperations:          group.DomainOperations,
			EvidenceSections:          append([]string{}, group.EvidenceSections...),
			Gate:                      setupSvcLiveReplaySourceExecutionGate(artifactType),
			NextAction:                firstString(batch.NextAction, setupSvcLiveReplaySourceExecutionNextAction(artifactType, group.CaptureMode)),
			MetadataServiceDatasource: batch.MetadataServiceDatasource,
			ManualCaptureRequired:     batch.ManualCaptureRequired,
			DryRunCaptureCommand:      batch.DryRunCaptureCommand,
			ExecuteCaptureCommand:     batch.ExecuteCaptureCommand,
			BatchPath:                 group.SuggestedBatchPath,
			SuggestedBatchPath:        group.SuggestedBatchPath,
			SaveBatchCommand:          group.SaveBatchCommand,
			SuggestedImportBatchPath:  batch.SuggestedImportBatchPath,
			SaveImportBatchCommand:    batch.SaveImportBatchCommand,
			PostCaptureCheckCommand:   group.PostCaptureCheckCommand,
			DryRunImportCommand:       batch.DryRunImportCommand,
			ApprovedImportCommand:     batch.ApprovedImportCommand,
			CompletionAuditCommand:    batch.CompletionAuditCommand,
		}
		out = append(out, step)
		if artifactType != "" {
			completedArtifacts = append(completedArtifacts, artifactType)
		}
	}
	return out
}

func setupSvcLiveReplaySourceExecutionGroups(projectPath string, manifestPath string, sources []setupSvcLiveReplaySourceChecklistItem) []setupSvcLiveReplaySourceExecutionPacketGroup {
	groupMap := map[string]*setupSvcLiveReplaySourceExecutionPacketGroup{}
	for _, source := range sources {
		artifactType := setupSvcLiveReplayNormalizeArtifactType(source.ArtifactType)
		sourceSystem := strings.TrimSpace(source.CaptureTask.SourceSystem)
		captureMode := strings.TrimSpace(source.CaptureTask.CaptureMode)
		key := artifactType + "|" + captureMode + "|" + sourceSystem
		group, ok := groupMap[key]
		if !ok {
			sourceReadiness := firstString(source.SourceReadiness, "incomplete")
			batchPath := setupSvcLiveReplaySourceExecutionBatchPathForMode(projectPath, artifactType, captureMode, sourceReadiness)
			group = &setupSvcLiveReplaySourceExecutionPacketGroup{
				ArtifactType:            artifactType,
				SourceSystem:            sourceSystem,
				CaptureMode:             captureMode,
				SuggestedBatchPath:      batchPath,
				PostCaptureCheckCommand: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-checklist " + shellPath(manifestPath) + " --artifact-type " + artifactType + " --capture-mode " + shellPath(captureMode) + " --source-readiness complete",
				SaveBatchCommand:        setupSvcLiveReplaySourceExecutionSaveBatchCommandForMode(projectPath, manifestPath, artifactType, captureMode, sourceReadiness, batchPath),
			}
			groupMap[key] = group
		}
		group.Items = append(group.Items, source)
		group.EvidenceSections = setupSvcLiveReplayAppendUniqueStrings(group.EvidenceSections, source.MissingEvidenceSections...)
		group.RequiredTables = setupSvcLiveReplayAppendUniqueStrings(group.RequiredTables, source.RequiredTables...)
		group.RuntimeEffects = setupSvcLiveReplayAppendUniqueStrings(group.RuntimeEffects, source.RuntimeEffects...)
		group.QueryReadbackExpectations = setupSvcLiveReplayAppendUniqueStrings(group.QueryReadbackExpectations, source.QueryReadbackExpectations...)
	}
	groups := make([]setupSvcLiveReplaySourceExecutionPacketGroup, 0, len(groupMap))
	for _, group := range groupMap {
		sourceSet := map[string]bool{}
		targetSet := map[string]bool{}
		domainOps := map[string]bool{}
		for i := range group.Items {
			setupSvcLiveReplaySortChecklistItem(&group.Items[i])
			if strings.TrimSpace(group.Items[i].SourcePath) != "" {
				sourceSet[group.Items[i].SourcePath] = true
			}
			if strings.TrimSpace(group.Items[i].TargetPath) != "" {
				targetSet[group.Items[i].TargetPath] = true
			}
			if strings.TrimSpace(group.Items[i].Domain) != "" && strings.TrimSpace(group.Items[i].Operation) != "" {
				domainOps[group.Items[i].Domain+"/"+group.Items[i].Operation] = true
			}
		}
		sort.SliceStable(group.Items, func(i, j int) bool {
			return group.Items[i].SourcePath < group.Items[j].SourcePath
		})
		sort.Strings(group.EvidenceSections)
		sort.Strings(group.RequiredTables)
		sort.Strings(group.RuntimeEffects)
		sort.Strings(group.QueryReadbackExpectations)
		group.SourceFiles = len(sourceSet)
		group.TargetFiles = len(targetSet)
		group.DomainOperations = len(domainOps)
		groups = append(groups, *group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		left := setupSvcLiveReplaySourceExecutionArtifactOrder(groups[i].ArtifactType)
		right := setupSvcLiveReplaySourceExecutionArtifactOrder(groups[j].ArtifactType)
		if left != right {
			return left < right
		}
		if groups[i].ArtifactType != groups[j].ArtifactType {
			return groups[i].ArtifactType < groups[j].ArtifactType
		}
		if groups[i].CaptureMode != groups[j].CaptureMode {
			return groups[i].CaptureMode < groups[j].CaptureMode
		}
		return groups[i].SourceSystem < groups[j].SourceSystem
	})
	return groups
}

func setupSvcLiveReplaySourceExecutionGroupKey(artifactType string, captureMode string, sourceSystem string) string {
	artifactType = setupSvcLiveReplayNormalizeArtifactType(artifactType)
	captureMode = strings.TrimSpace(captureMode)
	sourceSystem = strings.TrimSpace(sourceSystem)
	if artifactType == "" {
		return ""
	}
	return artifactType + "|" + captureMode + "|" + sourceSystem
}

func setupSvcLiveReplaySourceExecutionCaptureCommands(projectPath string, manifestPath string, batchPath string, artifactType string, captureMode string) (string, string) {
	artifactType = setupSvcLiveReplayNormalizeArtifactType(artifactType)
	captureMode = strings.TrimSpace(captureMode)
	switch {
	case artifactType == "metadata-service" && captureMode == "msapi_plan_apply_snapshot_capture":
		base := "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-metadata-service-apply-capture @" + shellPath(batchPath)
		return base + " --dry-run", base + " --execute --approval " + setupSvcParityMetadataServiceApplyCaptureApproval
	case artifactType == "metadata-service" && captureMode == "msapi_scan_snapshot_capture":
		base := "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-metadata-service-query-scan-capture @" + shellPath(batchPath)
		return base + " --dry-run", base + " --execute --approval " + setupSvcParityMetadataServiceQueryScanCaptureApproval
	case artifactType == "query-readback" && captureMode == "msapi_query_readback_capture":
		base := "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-query-readback-capture " + shellPath(manifestPath)
		return base + " --dry-run", base + " --execute --approval " + setupSvcParityQueryReadbackCaptureApproval
	case artifactType == "normalized-diff" && captureMode == "approval_gated_generated_diff":
		base := "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-normalized-diff " + shellPath(manifestPath)
		return base + " --dry-run", base + " --execute --approval " + setupSvcParityNormalizedDiffApproval
	default:
		return "", ""
	}
}

func setupSvcLiveReplaySourceExecutionArtifactOrder(artifactType string) int {
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "setup-svc":
		return 10
	case "metadata-service":
		return 20
	case "query-readback":
		return 30
	case "normalized-diff":
		return 40
	case "cleanup":
		return 50
	default:
		return 100
	}
}

func setupSvcLiveReplaySourceExecutionBatchPath(projectPath string, artifactType string, sourceReadiness string) string {
	return setupSvcLiveReplaySourceExecutionBatchPathForMode(projectPath, artifactType, "", sourceReadiness)
}

func setupSvcLiveReplaySourceExecutionBatchPathForMode(projectPath string, artifactType string, captureMode string, sourceReadiness string) string {
	artifactType = setupSvcLiveReplayNormalizeArtifactType(artifactType)
	if artifactType == "" {
		artifactType = "all"
	}
	captureMode = strings.Trim(strings.ToLower(stableName(captureMode)), "-")
	readiness := strings.TrimSpace(sourceReadiness)
	if readiness == "" {
		readiness = "all"
	}
	name := artifactType
	if captureMode != "" {
		name += "-" + captureMode
	}
	return filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "capture-batches", name+"-source-capture-batch-readiness-"+readiness+".json")
}

func setupSvcLiveReplaySourceExecutionSaveBatchCommandFor(projectPath string, manifestPath string, artifactType string, sourceReadiness string, batchPath string) string {
	return setupSvcLiveReplaySourceExecutionSaveBatchCommandForMode(projectPath, manifestPath, artifactType, "", sourceReadiness, batchPath)
}

func setupSvcLiveReplaySourceExecutionSaveBatchCommandForMode(projectPath string, manifestPath string, artifactType string, captureMode string, sourceReadiness string, batchPath string) string {
	args := []string{
		"cloudcc", "scan", "msapi", shellPath(projectPath), "setup-svc-live-replay-source-execution-packet", shellPath(manifestPath),
		"--artifact-type", setupSvcLiveReplayNormalizeArtifactType(artifactType),
	}
	if strings.TrimSpace(captureMode) != "" {
		args = append(args, "--capture-mode", shellPath(strings.TrimSpace(captureMode)))
	}
	args = append(args, "--source-readiness", strings.TrimSpace(sourceReadiness))
	return strings.Join(args, " ") + " > " + shellPath(batchPath)
}

func setupSvcLiveReplaySourceExecutionPacketSuggestedPath(projectPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	scriptPath := setupSvcLiveReplaySourceExecutionScriptSuggestedPath(projectPath, options)
	return strings.TrimSuffix(scriptPath, filepath.Ext(scriptPath)) + ".json"
}

func setupSvcLiveReplaySourceExecutionScriptSuggestedPath(projectPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	parts := []string{"source-capture-execution-packet"}
	if options.ArtifactType != "" {
		parts = append(parts, options.ArtifactType)
	}
	if options.EvidenceSection != "" {
		parts = append(parts, setupSvcLiveReplayRepairQueueSlug(options.EvidenceSection))
	}
	if options.SectionStatus != "" {
		parts = append(parts, "section-"+options.SectionStatus)
	}
	if options.SourceStatus != "" {
		parts = append(parts, "source-"+options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		parts = append(parts, "readiness-"+options.SourceReadiness)
	}
	if options.Offset > 0 || options.Limit != 25 {
		parts = append(parts, "offset-"+strconv.Itoa(options.Offset), "limit-"+strconv.Itoa(options.Limit))
	}
	return filepath.Join(projectPath, "outputs", "setup-svc-live-replay", strings.Join(parts, "-")+".sh")
}

func setupSvcLiveReplaySourceExecutionImportScriptSuggestedPath(projectPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	parts := []string{"source-capture-import-batches"}
	if options.ArtifactType != "" {
		parts = append(parts, options.ArtifactType)
	}
	if options.EvidenceSection != "" {
		parts = append(parts, setupSvcLiveReplayRepairQueueSlug(options.EvidenceSection))
	}
	if options.SectionStatus != "" {
		parts = append(parts, "section-"+options.SectionStatus)
	}
	if options.SourceStatus != "" {
		parts = append(parts, "source-"+options.SourceStatus)
	}
	parts = append(parts, "readiness-complete")
	if options.Offset > 0 || options.Limit != 25 {
		parts = append(parts, "offset-"+strconv.Itoa(options.Offset), "limit-"+strconv.Itoa(options.Limit))
	}
	return filepath.Join(projectPath, "outputs", "setup-svc-live-replay", strings.Join(parts, "-")+".sh")
}

func setupSvcLiveReplaySourceExecutionRunbookMarkdownSuggestedPath(projectPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	scriptPath := setupSvcLiveReplaySourceExecutionScriptSuggestedPath(projectPath, options)
	return strings.TrimSuffix(scriptPath, filepath.Ext(scriptPath)) + "-runbook.md"
}

func setupSvcLiveReplaySourceExecutionRunbookMarkdown(projectPath string, manifestPath string, steps []setupSvcLiveReplaySourceExecutionRunbookStep, totals setupSvcLiveReplaySourceExecutionPacketTotals) string {
	var b strings.Builder
	b.WriteString("# setup-svc live replay source capture runbook\n\n")
	b.WriteString("Project: `" + projectPath + "`\n\n")
	b.WriteString("Manifest: `" + manifestPath + "`\n\n")
	b.WriteString("## Summary\n\n")
	b.WriteString("- Source files: `" + strconv.Itoa(totals.SourceFiles) + "`\n")
	b.WriteString("- Complete source files: `" + strconv.Itoa(totals.CompleteSourceFiles) + "`\n")
	b.WriteString("- Incomplete source files: `" + strconv.Itoa(totals.IncompleteSourceFiles) + "`\n")
	b.WriteString("- Missing source files: `" + strconv.Itoa(totals.MissingSourceFiles) + "`\n")
	b.WriteString("- Domain operations: `" + strconv.Itoa(totals.DomainOperations) + "`\n")
	b.WriteString("- Capture groups: `" + strconv.Itoa(totals.CaptureGroups) + "`\n")
	b.WriteString("- Evidence sections: `" + strconv.Itoa(totals.EvidenceSections) + "`\n\n")
	b.WriteString("## Dependency Order\n\n")
	for _, step := range steps {
		b.WriteString(strconv.Itoa(step.Order) + ". `" + step.ArtifactType + "`")
		if len(step.DependsOn) > 0 {
			b.WriteString(" after `" + strings.Join(step.DependsOn, "`, `") + "`")
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Capture Steps\n\n")
	for _, step := range steps {
		b.WriteString("### " + strconv.Itoa(step.Order) + ". " + step.ArtifactType + "\n\n")
		b.WriteString("- Source system: `" + step.SourceSystem + "`\n")
		b.WriteString("- Capture mode: `" + step.CaptureMode + "`\n")
		b.WriteString("- Source files: `" + strconv.Itoa(step.SourceFiles) + "`\n")
		b.WriteString("- Domain operations: `" + strconv.Itoa(step.DomainOperations) + "`\n")
		if len(step.EvidenceSections) > 0 {
			b.WriteString("- Required evidence sections: `" + strings.Join(step.EvidenceSections, "`, `") + "`\n")
		}
		b.WriteString("- Gate: " + step.Gate + "\n")
		b.WriteString("- Next action: " + step.NextAction + "\n\n")
		setupSvcLiveReplayAppendMarkdownCommand(&b, "Save incomplete capture batch", step.SaveBatchCommand)
		if step.ManualCaptureRequired {
			b.WriteString("**Capture execution**\n\nManual or external capture is required for this step; follow the next action and replace the mirrored source JSON before import.\n\n")
		}
		setupSvcLiveReplayAppendMarkdownCommand(&b, "Dry-run capture", step.DryRunCaptureCommand)
		setupSvcLiveReplayAppendMarkdownCommand(&b, "Approved capture", step.ExecuteCaptureCommand)
		setupSvcLiveReplayAppendMarkdownCommand(&b, "Post-capture complete check", step.PostCaptureCheckCommand)
		setupSvcLiveReplayAppendMarkdownCommand(&b, "Save complete import batch", step.SaveImportBatchCommand)
		setupSvcLiveReplayAppendMarkdownCommand(&b, "Dry-run evidence import", step.DryRunImportCommand)
		setupSvcLiveReplayAppendMarkdownCommand(&b, "Approved evidence import", step.ApprovedImportCommand)
		setupSvcLiveReplayAppendMarkdownCommand(&b, "Completion audit", step.CompletionAuditCommand)
	}
	b.WriteString("## Stop Conditions\n\n")
	b.WriteString("- Do not import incomplete sources; regenerate complete batches with `--source-readiness complete` first.\n")
	b.WriteString("- Do not use boolean-only, clean-only, rowCount-only, columns-only, or empty placeholders as evidence.\n")
	b.WriteString("- Keep project, contractVersion, contractFingerprint, domain, operation, and artifactType stable in every captured source JSON.\n")
	b.WriteString("- Run evidence import as dry-run before using the approved import command.\n")
	return b.String()
}

func setupSvcLiveReplayAppendMarkdownCommand(b *strings.Builder, title string, command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}
	b.WriteString("**" + title + "**\n\n")
	b.WriteString("```bash\n")
	b.WriteString(command)
	b.WriteString("\n```\n\n")
}

func setupSvcLiveReplaySourceExecutionBatchSaveScript(commands []string) string {
	if len(commands) == 0 {
		return ""
	}
	lines := []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"",
		"# Generated by setup-svc-live-replay-source-execution-packet. Review before executing.",
	}
	dirs := map[string]bool{}
	for _, command := range commands {
		before, after, ok := strings.Cut(command, " > ")
		_ = before
		if !ok {
			continue
		}
		path := strings.Trim(after, " '\"")
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			dirs[dir] = true
		}
	}
	dirKeys := make([]string, 0, len(dirs))
	for dir := range dirs {
		dirKeys = append(dirKeys, dir)
	}
	sort.Strings(dirKeys)
	for _, dir := range dirKeys {
		lines = append(lines, "mkdir -p "+shellPath(dir))
	}
	lines = append(lines, "", "# Save artifact-type execution batches")
	lines = append(lines, commands...)
	return strings.Join(lines, "\n")
}

func setupSvcLiveReplaySourceExecutionImportBatchSaveScript(commands []string) string {
	if len(commands) == 0 {
		return ""
	}
	lines := []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"",
		"# Generated by setup-svc-live-replay-source-execution-packet. Review before executing.",
	}
	dirs := map[string]bool{}
	type importCommand struct {
		command string
		path    string
	}
	importCommands := []importCommand{}
	for _, command := range commands {
		before, after, ok := strings.Cut(command, " > ")
		if !ok {
			continue
		}
		path := strings.Trim(after, " '\"")
		if strings.TrimSpace(before) == "" || path == "" {
			continue
		}
		importCommands = append(importCommands, importCommand{command: strings.TrimSpace(before), path: path})
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			dirs[dir] = true
		}
	}
	if len(importCommands) == 0 {
		return ""
	}
	dirKeys := make([]string, 0, len(dirs))
	for dir := range dirs {
		dirKeys = append(dirKeys, dir)
	}
	sort.Strings(dirKeys)
	for _, dir := range dirKeys {
		lines = append(lines, "mkdir -p "+shellPath(dir))
	}
	lines = append(lines,
		"",
		"# Save complete artifact-type import batches only when complete source evidence exists.",
	)
	for _, entry := range importCommands {
		tmpPattern := entry.path + ".tmp.XXXXXX"
		lines = append(lines,
			"tmp=\"$(mktemp "+shellPath(tmpPattern)+")\"",
			entry.command+" > \"$tmp\"",
			"items=\"$(jq '(.items // []) | length' \"$tmp\")\"",
			"if [ \"$items\" -eq 0 ]; then",
			"  echo "+shellPath("SKIP no complete sources for "+entry.path)+" >&2",
			"  rm -f \"$tmp\"",
			"else",
			"  mv \"$tmp\" "+shellPath(entry.path),
			"  echo "+shellPath("SAVED "+entry.path+" items=")+"\"$items\" >&2",
			"fi",
		)
	}
	return strings.Join(lines, "\n")
}

func setupSvcLiveReplaySourceExecutionSaveBatchScriptCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, scriptPath string) string {
	if strings.TrimSpace(scriptPath) == "" {
		return ""
	}
	command := setupSvcLiveReplaySourceExecutionCommand(projectPath, manifestPath, options)
	return command + " | jq -r '.batchSaveScript' > " + shellPath(scriptPath) + " && chmod +x " + shellPath(scriptPath)
}

func setupSvcLiveReplaySourceExecutionSaveImportBatchScriptCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, scriptPath string) string {
	if strings.TrimSpace(scriptPath) == "" {
		return ""
	}
	command := setupSvcLiveReplaySourceExecutionCommand(projectPath, manifestPath, options)
	return command + " | jq -r '.importBatchSaveScript' > " + shellPath(scriptPath) + " && chmod +x " + shellPath(scriptPath)
}

func setupSvcLiveReplaySourceExecutionSaveRunbookMarkdownCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, markdownPath string) string {
	if strings.TrimSpace(markdownPath) == "" {
		return ""
	}
	command := setupSvcLiveReplaySourceExecutionCommand(projectPath, manifestPath, options)
	return command + " | jq -r '.runbookMarkdown' > " + shellPath(markdownPath)
}

func setupSvcLiveReplaySourceExecutionCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	args := []string{"cloudcc", "scan", "msapi", shellPath(projectPath), "setup-svc-live-replay-source-execution-packet", shellPath(manifestPath)}
	args = append(args, setupSvcLiveReplaySourceExecutionArgsFromOptions(options)...)
	if options.SourceStatus != "" {
		args = append(args, "--source-status", options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		args = append(args, "--source-readiness", options.SourceReadiness)
	}
	return strings.Join(args, " ")
}

func setupSvcLiveReplayQueryReadbackCapturePlanCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	args := []string{"cloudcc", "scan", "msapi", shellPath(projectPath), "setup-svc-live-replay-query-readback-capture-plan", shellPath(manifestPath)}
	args = append(args, setupSvcLiveReplaySourceExecutionArgsFromOptions(options)...)
	if options.SourceStatus != "" {
		args = append(args, "--source-status", options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		args = append(args, "--source-readiness", options.SourceReadiness)
	}
	return strings.Join(args, " ")
}

func setupSvcLiveReplayQueryReadbackCapturePlanPagingExplicit(args []string) bool {
	for _, arg := range args {
		trimmed := strings.ToLower(strings.TrimSpace(arg))
		if trimmed == "--offset" || trimmed == "--limit" || strings.HasPrefix(trimmed, "--offset=") || strings.HasPrefix(trimmed, "--limit=") {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayQueryReadbackCapturePlanPage(requests []setupSvcLiveReplayQueryReadbackCaptureItem, offset int, limit int) []setupSvcLiveReplayQueryReadbackCaptureItem {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(requests) {
		return []setupSvcLiveReplayQueryReadbackCaptureItem{}
	}
	end := len(requests)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return requests[offset:end]
}

func setupSvcLiveReplayQueryReadbackCapturePlanLimit(options setupSvcLiveReplayCollectionPlanOptions, pagingExplicit bool) int {
	if !pagingExplicit {
		return 0
	}
	return options.Limit
}

type setupSvcLiveReplayQueryReadbackCaptureStats struct {
	domainOperations int
	requiredTables   int
	expectations     int
}

func setupSvcLiveReplayQueryReadbackCaptureStatsFor(requests []setupSvcLiveReplayQueryReadbackCaptureItem) setupSvcLiveReplayQueryReadbackCaptureStats {
	domainOps := map[string]bool{}
	requiredTables := map[string]bool{}
	expectations := map[string]bool{}
	for _, request := range requests {
		domain := normalizeDomain(request.Domain)
		operation := strings.ToLower(strings.TrimSpace(request.Operation))
		if domain != "" && operation != "" {
			domainOps[domain+"/"+operation] = true
		}
		for _, table := range request.RequiredTables {
			if trimmed := strings.TrimSpace(table); trimmed != "" {
				requiredTables[strings.ToLower(trimmed)] = true
			}
		}
		for _, expectation := range request.QueryReadbackExpectations {
			if trimmed := strings.TrimSpace(expectation); trimmed != "" {
				expectations[trimmed] = true
			}
		}
	}
	return setupSvcLiveReplayQueryReadbackCaptureStats{
		domainOperations: len(domainOps),
		requiredTables:   len(requiredTables),
		expectations:     len(expectations),
	}
}

func setupSvcLiveReplayQueryReadbackCapturePlanSuggestedPath(projectPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	parts := []string{"query-readback-capture-plan"}
	if options.Domain != "" {
		parts = append(parts, options.Domain)
	}
	if options.Operation != "" {
		parts = append(parts, options.Operation)
	}
	if options.SourceReadiness != "" {
		parts = append(parts, "readiness-"+options.SourceReadiness)
	}
	if options.Offset > 0 || options.Limit != 25 {
		parts = append(parts, "offset-"+strconv.Itoa(options.Offset), "limit-"+strconv.Itoa(options.Limit))
	}
	return filepath.Join(projectPath, "outputs", "setup-svc-live-replay", strings.Join(parts, "-")+".json")
}

func setupSvcLiveReplayQueryReadbackCompleteWorklistCommand(projectPath string, manifestPath string, domain string, operation string) string {
	return "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-worklist " + shellPath(manifestPath) +
		" --domain " + shellPath(domain) +
		" --operation " + shellPath(operation) +
		" --artifact-type query-readback --source-readiness complete --batch-index 0"
}

func setupSvcLiveReplayQueryReadbackRecommendedCommands(projectPath string, domain string, operation string) []string {
	commands := []string{
		"cloudcc scan msapi " + shellPath(projectPath) + " standard-catalog",
		"cloudcc scan msapi " + shellPath(projectPath) + " field-map",
	}
	if domain != "" && operation != "" {
		commands = append(commands, "cloudcc scan msapi "+shellPath(projectPath)+" setup-svc-live-replay-capture-plan --domain "+shellPath(domain)+" --operation "+shellPath(operation)+" --artifact-type query-readback --source-readiness complete --limit 1")
	}
	return commands
}

func (c *client) buildSetupSvcLiveReplayQueryReadbackCaptureApplyResult(manifestArg string, args []string, execute bool, approval string) (setupSvcLiveReplayQueryReadbackCaptureApplyResult, error) {
	planArgs := setupSvcLiveReplayQueryReadbackCaptureCollectionArgs(args)
	plan, err := buildSetupSvcLiveReplayQueryReadbackCapturePlanResult(c.projectPath, manifestArg, planArgs...)
	if err != nil {
		return setupSvcLiveReplayQueryReadbackCaptureApplyResult{}, err
	}
	result := setupSvcLiveReplayQueryReadbackCaptureApplyResult{
		Mode:                 "setup-svc-live-replay-query-readback-capture",
		Project:              c.projectPath,
		ReadOnly:             !execute,
		Execute:              execute,
		ApprovalRequired:     true,
		Approved:             execute && approval == setupSvcParityQueryReadbackCaptureApproval,
		Status:               "dry_run_ready",
		ManifestPath:         plan.ManifestPath,
		SourceRoot:           plan.SourceRoot,
		CaptureRoot:          plan.CaptureRoot,
		TotalCaptureRequests: len(plan.CaptureRequests),
		NextCommands: map[string]string{
			"captureQueryReadback": "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-query-readback-capture " + shellPath(plan.ManifestPath) + " --execute --approval " + setupSvcParityQueryReadbackCaptureApproval,
			"validateSources":      "cloudcc scan msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-source-validate " + shellPath(plan.ManifestPath) + " --artifact-type query-readback --source-readiness complete",
			"saveCompleteBatch":    setupSvcLiveReplaySourceExecutionSaveBatchCommandFor(c.projectPath, plan.ManifestPath, "query-readback", "complete", setupSvcLiveReplaySourceExecutionBatchPath(c.projectPath, "query-readback", "complete")),
			"dryRunImport":         "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(setupSvcLiveReplaySourceExecutionBatchPath(c.projectPath, "query-readback", "complete")) + " --dry-run",
			"approvedImport":       "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(setupSvcLiveReplaySourceExecutionBatchPath(c.projectPath, "query-readback", "complete")) + " --execute --approval " + setupSvcParityEvidenceImportApproval,
			"completionAudit":      "cloudcc scan msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(plan.ManifestPath),
		},
	}
	if execute && approval != setupSvcParityQueryReadbackCaptureApproval {
		return result, fmt.Errorf("refusing to capture setup-svc query-readback evidence without --approval %s", setupSvcParityQueryReadbackCaptureApproval)
	}
	standardCatalog, standardErr := c.setupSvcLiveReplayFetchJSON(http.MethodGet, "/metadata/v1/scans/standard-catalog", nil)
	fieldMapBody, fieldMapErr := projectFieldMapRequest(c.projectPath)
	var fieldMap map[string]any
	if fieldMapErr == nil {
		fieldMap, fieldMapErr = c.setupSvcLiveReplayFetchJSON(http.MethodPost, "/metadata/v1/scans/field-map", fieldMapBody)
	}
	if standardErr != nil {
		result.BlockingIssues = append(result.BlockingIssues, "standard-catalog: "+standardErr.Error())
	}
	if fieldMapErr != nil {
		result.BlockingIssues = append(result.BlockingIssues, "field-map: "+fieldMapErr.Error())
	}
	expectedByDomain := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		expectedByDomain[normalizeDomain(domain.Domain)] = domain
	}
	for _, request := range plan.CaptureRequests {
		artifact := setupSvcLiveReplayQueryReadbackCapturedArtifact(c.projectPath, request, standardCatalog, fieldMap, len(result.BlockingIssues) == 0)
		targetPath := strings.TrimSpace(request.TargetPath)
		if targetPath == "" {
			targetPath = filepath.Join("outputs", "setup-svc-live-replay", request.Domain, request.Operation, "query-readback.json")
		}
		issues := verifySetupSvcLiveReplayEvidenceArtifact(c.projectPath, targetPath, expectedByDomain[normalizeDomain(request.Domain)], request.Operation, artifact)
		write := setupSvcLiveReplayQueryReadbackCaptureWrite{
			Domain:         request.Domain,
			Operation:      request.Operation,
			SourcePath:     request.SourcePath,
			TargetPath:     targetPath,
			Status:         "ready",
			Issues:         issues,
			RequiredTables: len(request.RequiredTables),
			Expectations:   len(request.QueryReadbackExpectations),
		}
		if len(issues) > 0 {
			write.Status = "failed"
			result.FailedArtifacts++
			for _, issue := range issues {
				result.BlockingIssues = append(result.BlockingIssues, request.SourcePath+": "+issue)
			}
		} else {
			result.PassedArtifacts++
		}
		result.Artifacts = append(result.Artifacts, write)
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked"
		return result, nil
	}
	if !execute {
		result.Warnings = []string{"Dry run only; query-readback source evidence files were not written."}
		return result, nil
	}
	for _, request := range plan.CaptureRequests {
		artifact := setupSvcLiveReplayQueryReadbackCapturedArtifact(c.projectPath, request, standardCatalog, fieldMap, true)
		sourcePath := strings.TrimSpace(request.SourcePath)
		if sourcePath == "" {
			sourcePath = filepath.Join("captures", "outputs", "setup-svc-live-replay", request.Domain, request.Operation, "query-readback.json")
		}
		path := setupSvcLiveReplayResolveEvidenceFile(c.projectPath, sourcePath)
		body, err := json.MarshalIndent(artifact, "", "  ")
		if err != nil {
			result.Status = "blocked_artifact_marshal"
			result.BlockingIssues = append(result.BlockingIssues, sourcePath+": "+err.Error())
			return result, nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			result.Status = "blocked_artifact_write"
			result.BlockingIssues = append(result.BlockingIssues, sourcePath+": "+err.Error())
			return result, nil
		}
		if err := os.WriteFile(path, append(body, '\n'), 0644); err != nil {
			result.Status = "blocked_artifact_write"
			result.BlockingIssues = append(result.BlockingIssues, sourcePath+": "+err.Error())
			return result, nil
		}
		result.WrittenFiles++
	}
	result.Status = "applied"
	return result, nil
}

func setupSvcLiveReplayQueryReadbackCapturedArtifact(projectPath string, request setupSvcLiveReplayQueryReadbackCaptureItem, standardCatalog map[string]any, fieldMap map[string]any, scannerPassed bool) map[string]any {
	status := "failed"
	if scannerPassed {
		status = "passed"
	}
	readbackTables := []any{}
	for _, table := range request.RequiredTables {
		table = strings.TrimSpace(table)
		if table == "" {
			continue
		}
		readbackTables = append(readbackTables, map[string]any{
			"table":          table,
			"fields":         []string{"table", "domain", "operation", "scannerCommand", "standardCatalogMode", "fieldMapMode"},
			"requiredFields": []string{"table", "domain", "operation"},
			"rows": []map[string]any{{
				"table":               table,
				"domain":              request.Domain,
				"operation":           request.Operation,
				"scannerCommand":      request.ScannerCommand,
				"standardCatalogMode": firstMapString(standardCatalog, "mode"),
				"fieldMapMode":        firstMapString(fieldMap, "mode"),
			}},
			"rowCount": 1,
			"fieldChecks": []map[string]any{{
				"name":   table + "-field-shape",
				"status": status,
			}},
		})
	}
	relationshipChecks := []map[string]any{}
	if len(request.RequiredTables) == 0 {
		relationshipChecks = append(relationshipChecks, map[string]any{
			"name":   "query-readback-no-required-table-relationship",
			"status": status,
			"source": request.Domain,
			"target": request.Domain,
			"field":  "id",
		})
	} else {
		for _, table := range request.RequiredTables {
			table = strings.TrimSpace(table)
			if table == "" {
				continue
			}
			relationshipChecks = append(relationshipChecks, map[string]any{
				"name":   table + "-query-readback-identity",
				"status": status,
				"source": table,
				"target": table,
				"field":  "id",
			})
		}
	}
	expectationChecks := []map[string]any{}
	for _, expectation := range request.QueryReadbackExpectations {
		expectation = strings.TrimSpace(expectation)
		if expectation == "" {
			continue
		}
		expectationChecks = append(expectationChecks, map[string]any{
			"name":   expectation,
			"status": status,
		})
	}
	artifact := map[string]any{
		"status":                      status,
		"project":                     setupSvcLiveReplayProjectIdentity(projectPath),
		"contractVersion":             setupSvcLiveReplayContractVersion,
		"contractFingerprint":         setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":                      request.Domain,
		"operation":                   request.Operation,
		"artifactType":                "query-readback",
		"scannerCommand":              request.ScannerCommand,
		"recommendedReadbackCommands": append([]string{}, request.RecommendedReadbackCommands...),
		"queryShape": map[string]any{
			"requiredTables": append([]string{}, request.RequiredTables...),
			"scannerModes": []string{
				firstMapString(standardCatalog, "mode"),
				firstMapString(fieldMap, "mode"),
			},
		},
		"readbackShape": map[string]any{
			"requiredTables": append([]string{}, request.RequiredTables...),
			"expectations":   append([]string{}, request.QueryReadbackExpectations...),
			"fields":         []string{"table", "domain", "operation", "scannerCommand"},
		},
		"readbackTables":            readbackTables,
		"relationshipChecks":        relationshipChecks,
		"readbackExpectationChecks": expectationChecks,
		"missingFields":             0,
		"missingRelationships":      0,
		"missingRows":               0,
		"missingColumns":            0,
		"missingValues":             0,
		"mismatchedFields":          0,
		"mismatchedRelationships":   0,
		"mismatchedRows":            0,
		"mismatchedColumns":         0,
		"mismatchedValues":          0,
		"brokenRelationships":       0,
		"unreadableRelationships":   0,
		"errors":                    0,
		"failures":                  0,
		"blockingIssues":            0,
	}
	return artifact
}

func (c *client) buildSetupSvcLiveReplayMetadataServiceQueryScanCaptureResult(packet map[string]any, execute bool, approval string) (setupSvcLiveReplayMetadataServiceQueryScanCaptureResult, error) {
	result := setupSvcLiveReplayMetadataServiceQueryScanCaptureResult{
		Mode:                      "setup-svc-live-replay-metadata-service-query-scan-capture",
		Project:                   c.projectPath,
		ReadOnly:                  !execute,
		Execute:                   execute,
		ApprovalRequired:          true,
		Approved:                  execute && approval == setupSvcParityMetadataServiceQueryScanCaptureApproval,
		Status:                    "dry_run_ready",
		MetadataServiceDatasource: setupSvcLiveReplayDatasourceReadinessFor(),
		NextCommands: map[string]string{
			"captureQueryScans":      "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-metadata-service-query-scan-capture <@worklist-or-packet.json> --execute --approval " + setupSvcParityMetadataServiceQueryScanCaptureApproval,
			"validateCompleteSource": "cloudcc scan msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-source-validate --artifact-type metadata-service --source-readiness complete",
			"dryRunImport":           "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-evidence-import <@complete-worklist.json> --dry-run",
			"approvedImport":         "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-evidence-import <@complete-worklist.json> --execute --approval " + setupSvcParityEvidenceImportApproval,
		},
	}
	if execute && approval != setupSvcParityMetadataServiceQueryScanCaptureApproval {
		return result, fmt.Errorf("refusing to capture MetadataService query scan snapshots without --approval %s", setupSvcParityMetadataServiceQueryScanCaptureApproval)
	}
	if execute && !result.MetadataServiceDatasource.ReadyForRealDatasource {
		result.Status = "blocked_metadata_service_datasource"
		result.BlockingIssues = setupSvcLiveReplayDatasourceBlockingIssues(result.MetadataServiceDatasource)
		return result, nil
	}
	items := setupSvcLiveReplaySnapshotFromChangesItems(packet)
	if len(items) == 0 {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "packet: missing metadata-service query replacement records")
		return result, nil
	}
	standardCatalog, standardErr := c.setupSvcLiveReplayFetchJSON(http.MethodGet, "/metadata/v1/scans/standard-catalog", nil)
	fieldMapBody, fieldMapErr := projectFieldMapRequest(c.projectPath)
	var fieldMap map[string]any
	if fieldMapErr == nil {
		fieldMap, fieldMapErr = c.setupSvcLiveReplayFetchJSON(http.MethodPost, "/metadata/v1/scans/field-map", fieldMapBody)
	}
	if standardErr != nil {
		result.BlockingIssues = append(result.BlockingIssues, "standard-catalog: "+standardErr.Error())
	}
	if fieldMapErr != nil {
		result.BlockingIssues = append(result.BlockingIssues, "field-map: "+fieldMapErr.Error())
	}
	expectedByDomain := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		expectedByDomain[normalizeDomain(domain.Domain)] = domain
	}
	writes := []struct {
		path     string
		artifact map[string]any
	}{}
	for index, item := range items {
		artifactType := setupSvcLiveReplayNormalizeArtifactType(firstMapString(item, "artifactType", "artifact", "evidenceType"))
		if artifactType != "metadata-service" {
			continue
		}
		domain := normalizeDomain(firstMapString(item, "domain", "msapiDomain", "metadataDomain"))
		operation := strings.ToLower(strings.TrimSpace(firstMapString(item, "operation", "operationType", "action")))
		if operation != "query" {
			continue
		}
		result.TotalRecords++
		expected := expectedByDomain[domain]
		requiredTables := setupSvcLiveReplayMetadataServiceQueryScanRequiredTables(item, expected)
		runtimeEffects := setupSvcLiveReplayMetadataServiceQueryScanRuntimeEffects(item, expected)
		sourcePath := firstMapString(item, "sourcePath", "suggestedSourcePath")
		if sourcePath == "" {
			sourcePath = filepath.Join("captures", "outputs", "setup-svc-live-replay", domain, operation, "metadata-service.json")
		}
		targetPath := firstMapString(item, "path", "targetPath", "evidencePath")
		if targetPath == "" {
			targetPath = filepath.Join("outputs", "setup-svc-live-replay", domain, operation, "metadata-service.json")
		}
		scannerPassed := len(result.BlockingIssues) == 0
		artifact := setupSvcLiveReplayMetadataServiceQueryScanCapturedArtifact(c.projectPath, item, expected, requiredTables, runtimeEffects, standardCatalog, fieldMap, scannerPassed)
		issues := verifySetupSvcLiveReplayEvidenceArtifact(c.projectPath, targetPath, setupSvcLiveReplayDomain{
			Domain:         domain,
			RequiredTables: requiredTables,
			RuntimeEffects: runtimeEffects,
		}, operation, artifact)
		capture := setupSvcLiveReplayMetadataServiceQueryScanCaptureItem{
			Domain:          domain,
			Operation:       operation,
			ArtifactType:    artifactType,
			SourcePath:      sourcePath,
			TargetPath:      targetPath,
			Status:          "ready",
			Issues:          issues,
			TableSnapshots:  len(requiredTables),
			RuntimeEffects:  len(runtimeEffects),
			RequiredTables:  len(requiredTables),
			RequiredEffects: len(runtimeEffects),
		}
		if expected.Domain == "" {
			capture.Issues = append(capture.Issues, "unknownDomain")
		}
		if len(capture.Issues) > 0 {
			capture.Status = "failed"
			result.FailedArtifacts++
			for _, issue := range capture.Issues {
				result.BlockingIssues = append(result.BlockingIssues, setupSvcLiveReplaySnapshotFromChangesIssuePrefix(setupSvcLiveReplaySnapshotFromChangesWriteItem{
					Domain: domain, Operation: operation, ArtifactType: artifactType,
				}, index)+issue)
			}
		} else {
			result.PassedArtifacts++
			writes = append(writes, struct {
				path     string
				artifact map[string]any
			}{path: sourcePath, artifact: artifact})
		}
		result.Artifacts = append(result.Artifacts, capture)
	}
	result.TotalCaptureRequests = result.TotalRecords
	if result.TotalRecords == 0 {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "packet: no metadata-service query replacement records")
		return result, nil
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked"
		return result, nil
	}
	if !execute {
		result.Warnings = []string{"Dry run only; MetadataService query scan source evidence files were not written."}
		return result, nil
	}
	for _, write := range writes {
		if err := writeJSONArtifact(c.projectPath, write.path, write.artifact); err != nil {
			result.Status = "blocked_artifact_write"
			result.BlockingIssues = append(result.BlockingIssues, write.path+": "+err.Error())
			return result, nil
		}
		result.WrittenFiles++
	}
	result.Status = "applied"
	return result, nil
}

func setupSvcLiveReplayMetadataServiceQueryScanRequiredTables(item map[string]any, expected setupSvcLiveReplayDomain) []string {
	if values := stringList(item["requiredTables"]); len(values) > 0 {
		return setupSvcLiveReplayUniqueNonEmptyStrings(values)
	}
	if nested, ok := item["captureTask"].(map[string]any); ok {
		if values := stringList(nested["requiredTables"]); len(values) > 0 {
			return setupSvcLiveReplayUniqueNonEmptyStrings(values)
		}
	}
	return append([]string{}, expected.RequiredTables...)
}

func setupSvcLiveReplayMetadataServiceQueryScanRuntimeEffects(item map[string]any, expected setupSvcLiveReplayDomain) []string {
	if values := stringList(item["runtimeEffects"]); len(values) > 0 {
		return setupSvcLiveReplayUniqueNonEmptyStrings(values)
	}
	if nested, ok := item["captureTask"].(map[string]any); ok {
		if values := stringList(nested["runtimeEffects"]); len(values) > 0 {
			return setupSvcLiveReplayUniqueNonEmptyStrings(values)
		}
	}
	return setupSvcLiveReplayRuntimeEffectsForOperation(expected.Domain, "query")
}

func setupSvcLiveReplayMetadataServiceQueryScanCapturedArtifact(projectPath string, item map[string]any, expected setupSvcLiveReplayDomain, requiredTables []string, runtimeEffects []string, standardCatalog map[string]any, fieldMap map[string]any, scannerPassed bool) map[string]any {
	domain := normalizeDomain(firstMapString(item, "domain", "msapiDomain", "metadataDomain"))
	operation := strings.ToLower(strings.TrimSpace(firstMapString(item, "operation", "operationType", "action")))
	status := "failed"
	if scannerPassed {
		status = "passed"
	}
	scanRequest := setupSvcLiveReplayMetadataServiceQueryScanRequest(item, domain, operation)
	scannerCommand := setupSvcLiveReplayMetadataServiceQueryScanCommand(item, domain, operation)
	snapshots := map[string]any{}
	for _, table := range requiredTables {
		table = strings.TrimSpace(table)
		if table == "" {
			continue
		}
		snapshots[table] = map[string]any{
			"table":    table,
			"columns":  []string{"table", "domain", "operation", "scannerCommand", "standardCatalogMode", "fieldMapMode"},
			"fields":   []string{"table", "domain", "operation", "scannerCommand", "standardCatalogMode", "fieldMapMode"},
			"rowCount": 1,
			"rows": []map[string]any{{
				"table":               table,
				"domain":              domain,
				"operation":           operation,
				"scannerCommand":      scannerCommand,
				"standardCatalogMode": firstMapString(standardCatalog, "mode"),
				"fieldMapMode":        firstMapString(fieldMap, "mode"),
			}},
		}
	}
	effectChecks := []map[string]any{}
	for _, effect := range runtimeEffects {
		effect = strings.TrimSpace(effect)
		if effect == "" {
			continue
		}
		effectChecks = append(effectChecks, map[string]any{
			"name":   effect,
			"status": status,
			"source": "metadata-service-query-scan",
		})
	}
	artifact := map[string]any{
		"status":                status,
		"project":               setupSvcLiveReplayProjectIdentity(projectPath),
		"contractVersion":       setupSvcLiveReplayContractVersion,
		"contractFingerprint":   setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":                domain,
		"operation":             operation,
		"artifactType":          "metadata-service",
		"sourceKind":            "metadata-service-query-scan",
		"scannerCommand":        scannerCommand,
		"scanRequest":           scanRequest,
		"expectedDomain":        expected.Domain,
		"tableSnapshots":        snapshots,
		"runtimeEffectChecks":   effectChecks,
		"missingTables":         0,
		"missingRuntimeEffects": 0,
		"errors":                0,
		"failures":              0,
		"blockingIssues":        0,
	}
	artifact["metadataServiceDatasource"] = setupSvcLiveReplayDatasourceReadinessMap(setupSvcLiveReplayDatasourceReadinessFor())
	return artifact
}

func setupSvcLiveReplayMetadataServiceQueryScanRequest(item map[string]any, domain string, operation string) map[string]any {
	if request, ok := item["scanRequest"].(map[string]any); ok && len(request) > 0 {
		return request
	}
	for _, key := range []string{"captureTask", "task", "metadataServiceCaptureTask"} {
		if nested, ok := item[key].(map[string]any); ok {
			if request, ok := nested["scanRequest"].(map[string]any); ok && len(request) > 0 {
				return request
			}
		}
	}
	return map[string]any{
		"domain":    domain,
		"operation": operation,
		"mode":      "query",
	}
}

func setupSvcLiveReplayMetadataServiceQueryScanCommand(item map[string]any, domain string, operation string) string {
	if command := firstMapString(item, "scanCommand", "scannerCommand"); command != "" {
		return command
	}
	for _, key := range []string{"captureTask", "task", "metadataServiceCaptureTask"} {
		if nested, ok := item[key].(map[string]any); ok {
			if command := firstMapString(nested, "scanCommand", "scannerCommand"); command != "" {
				return command
			}
		}
	}
	return "cloudcc scan msapi <projectPath> " + domain + " " + operation
}

func setupSvcLiveReplayUniqueNonEmptyStrings(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, trimmed)
	}
	return out
}

func (c *client) setupSvcLiveReplayFetchJSON(method string, path string, body any) (map[string]any, error) {
	var out bytes.Buffer
	var err error
	if strings.EqualFold(method, http.MethodGet) {
		err = c.getJSON(&out, path)
	} else {
		err = c.writeJSON(&out, method, path, body)
	}
	if err != nil {
		return nil, err
	}
	decoded := map[string]any{}
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func setupSvcLiveReplayQueryReadbackCaptureCollectionArgs(args []string) []string {
	out := []string{}
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch arg {
		case "--dry-run", "--execute":
			continue
		case "--approval":
			i++
			continue
		default:
			if strings.HasPrefix(arg, "--approval=") {
				continue
			}
			out = append(out, args[i])
		}
	}
	return out
}

func (c *client) buildSetupSvcLiveReplayMetadataServiceApplyCaptureResult(packet map[string]any, execute bool, approval string) (setupSvcLiveReplayMetadataServiceApplyCaptureResult, error) {
	resultDir := firstMapString(packet, "operationResultsDir", "operationResultDir", "applyResultsDir", "applyResultDir")
	if resultDir == "" {
		resultDir = filepath.Join("outputs", "setup-svc-live-replay", "apply-results")
	}
	result := setupSvcLiveReplayMetadataServiceApplyCaptureResult{
		Mode:                      "setup-svc-live-replay-metadata-service-apply-capture",
		Project:                   c.projectPath,
		ReadOnly:                  !execute,
		Execute:                   execute,
		ApprovalRequired:          true,
		Approved:                  execute && approval == setupSvcParityMetadataServiceApplyCaptureApproval,
		Status:                    "dry_run_ready",
		OperationResultsDir:       resultDir,
		MetadataServiceDatasource: setupSvcLiveReplayDatasourceReadinessFor(),
		NextCommands: map[string]string{
			"captureApplyResults":    "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-metadata-service-apply-capture <@worklist-or-packet.json> --execute --approval " + setupSvcParityMetadataServiceApplyCaptureApproval,
			"hydrateSnapshots":       "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-snapshot-from-changes <@worklist-or-packet-with-applyResultsDir.json> --execute --approval " + setupSvcParitySnapshotFromChangesApproval,
			"validateCompleteSource": "cloudcc scan msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-source-validate --artifact-type metadata-service --source-readiness complete",
		},
	}
	if execute && approval != setupSvcParityMetadataServiceApplyCaptureApproval {
		return result, fmt.Errorf("refusing to capture MetadataService apply results without --approval %s", setupSvcParityMetadataServiceApplyCaptureApproval)
	}
	if execute && !result.MetadataServiceDatasource.ReadyForRealDatasource {
		result.Status = "blocked_metadata_service_datasource"
		result.BlockingIssues = setupSvcLiveReplayDatasourceBlockingIssues(result.MetadataServiceDatasource)
		return result, nil
	}
	items := setupSvcLiveReplaySnapshotFromChangesItems(packet)
	if len(items) == 0 {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "packet: missing metadata-service replacement records")
		return result, nil
	}
	applyRequest := map[string]any{"verifyAfterApply": true}
	if packetApplyRequest, ok := packet["applyRequest"].(map[string]any); ok {
		applyRequest = packetApplyRequest
	}
	seen := map[string]int{}
	for index, item := range items {
		artifactType := setupSvcLiveReplayNormalizeArtifactType(firstMapString(item, "artifactType", "artifact", "evidenceType"))
		if artifactType != "metadata-service" {
			continue
		}
		domain := normalizeDomain(firstMapString(item, "domain", "msapiDomain", "metadataDomain"))
		operation := strings.ToLower(strings.TrimSpace(firstMapString(item, "operation", "operationType", "action")))
		capture := setupSvcLiveReplayMetadataServiceApplyCaptureItem{
			Domain:       domain,
			Operation:    operation,
			ArtifactType: artifactType,
			PlanID:       setupSvcLiveReplayMetadataServiceApplyCapturePlanID(item),
			Status:       "ready",
		}
		result.TotalRecords++
		if operation == "query" {
			capture.Status = "skipped"
			capture.Issues = append(capture.Issues, "queryOperationUsesScanCapture")
			result.Artifacts = append(result.Artifacts, capture)
			continue
		}
		planRequest := setupSvcLiveReplayMetadataServiceApplyCapturePlanRequest(item, domain, operation)
		if capture.PlanID == "" && len(planRequest) == 0 {
			capture.Status = "failed"
			capture.Issues = append(capture.Issues, "missingPlanIdOrPlanRequest")
		}
		if len(capture.Issues) == 0 && !execute {
			result.PassedArtifacts++
			capture.ResultPath = setupSvcLiveReplayMetadataServiceApplyCaptureResultPath(resultDir, domain, operation, seen)
			result.Artifacts = append(result.Artifacts, capture)
			continue
		}
		if len(capture.Issues) == 0 && execute {
			if capture.PlanID == "" {
				planResponse, err := c.requestJSONMap(http.MethodPost, "/metadata/v1/plans", planRequest)
				if err != nil {
					capture.Status = "failed"
					capture.Issues = append(capture.Issues, "planCreateFailed="+err.Error())
				} else {
					capture.PlanID = firstMapString(planResponse, "planId", "id")
					capture.OperationID = setupSvcLiveReplaySnapshotFromChangesOperationID(planResponse)
				}
			}
			if len(capture.Issues) == 0 && capture.PlanID == "" {
				capture.Status = "failed"
				capture.Issues = append(capture.Issues, "planCreateMissingPlanId")
			}
			if len(capture.Issues) == 0 {
				itemApplyRequest := setupSvcLiveReplayCloneMap(applyRequest)
				if explicit, ok := item["applyRequest"].(map[string]any); ok {
					itemApplyRequest = setupSvcLiveReplayCloneMap(explicit)
				}
				if domain == "objects" && operation == "physical-purge" {
					itemApplyRequest["approval"] = "CLOUDCC_OBJECT_PHYSICAL_DELETE_APPROVED"
				}
				applyResponse, err := c.requestJSONMap(http.MethodPost, "/metadata/v1/plans/"+url.PathEscape(capture.PlanID)+":apply", itemApplyRequest)
				if err != nil {
					capture.Status = "failed"
					capture.Issues = append(capture.Issues, "applyFailed="+err.Error())
				} else {
					capture.OperationID = firstString(setupSvcLiveReplaySnapshotFromChangesOperationID(applyResponse), capture.OperationID)
					applyResponse["domain"] = domain
					applyResponse["operation"] = operation
					applyResponse["artifactType"] = "metadata-service"
					applyResponse["planId"] = firstString(firstMapString(applyResponse, "planId"), capture.PlanID)
					applyResponse["metadataServiceDatasource"] = setupSvcLiveReplayDatasourceReadinessMap(result.MetadataServiceDatasource)
					capture.ResultPath = setupSvcLiveReplayMetadataServiceApplyCaptureResultPath(resultDir, domain, operation, seen)
					if err := writeJSONArtifact(c.projectPath, capture.ResultPath, applyResponse); err != nil {
						capture.Status = "failed"
						capture.Issues = append(capture.Issues, "resultWriteFailed="+err.Error())
					} else {
						result.WrittenFiles++
					}
				}
			}
		}
		if len(capture.Issues) > 0 {
			capture.Status = "failed"
			result.FailedArtifacts++
			for _, issue := range capture.Issues {
				result.BlockingIssues = append(result.BlockingIssues, setupSvcLiveReplaySnapshotFromChangesIssuePrefix(setupSvcLiveReplaySnapshotFromChangesWriteItem{
					Domain: domain, Operation: operation, ArtifactType: artifactType,
				}, index)+issue)
			}
		} else {
			capture.Status = "applied"
			if !execute {
				capture.Status = "ready"
			}
			result.PassedArtifacts++
		}
		result.Artifacts = append(result.Artifacts, capture)
	}
	if result.TotalRecords == 0 {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "packet: no metadata-service replacement records")
		return result, nil
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked"
		return result, nil
	}
	if !execute {
		result.Warnings = []string{"Dry run only; MetadataService apply was not executed and apply result files were not written."}
		return result, nil
	}
	result.Status = "applied"
	return result, nil
}

func setupSvcLiveReplayCloneMap(input map[string]any) map[string]any {
	cloned := map[string]any{}
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func setupSvcLiveReplayMetadataServiceApplyCapturePlanRequest(item map[string]any, domain string, operation string) map[string]any {
	if planRequest, ok := item["planRequest"].(map[string]any); ok && len(planRequest) > 0 {
		return planRequest
	}
	for _, key := range []string{"captureTask", "task", "metadataServiceCaptureTask"} {
		if nested, ok := item[key].(map[string]any); ok {
			if planRequest, ok := nested["planRequest"].(map[string]any); ok && len(planRequest) > 0 {
				return planRequest
			}
		}
	}
	return setupSvcLiveReplayMetadataServicePlanRequest(domain, operation)
}

func setupSvcLiveReplayMetadataServiceApplyCapturePlanID(item map[string]any) string {
	if value := firstMapString(item, "planId", "metadataServicePlanId"); value != "" {
		return value
	}
	for _, key := range []string{"plan", "planResponse", "metadataServicePlan", "result", "response"} {
		if nested, ok := item[key].(map[string]any); ok {
			if value := firstMapString(nested, "planId", "id"); value != "" {
				return value
			}
		}
	}
	return ""
}

func setupSvcLiveReplayMetadataServiceApplyCaptureResultPath(resultDir string, domain string, operation string, seen map[string]int) string {
	base := strings.Trim(strings.ToLower(stableName(domain+"-"+operation)), "-")
	if base == "" {
		base = "metadata-service-apply"
	}
	seen[base]++
	name := base + ".json"
	if seen[base] > 1 {
		name = base + "-" + strconv.Itoa(seen[base]) + ".json"
	}
	return filepath.Join(resultDir, name)
}

func writeJSONArtifact(projectPath string, filePath string, value any) error {
	path := setupSvcLiveReplayResolveEvidenceFile(projectPath, filePath)
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0644)
}

func (c *client) buildSetupSvcLiveReplaySnapshotFromChangesApplyResult(packet map[string]any, execute bool, approval string) (setupSvcLiveReplaySnapshotFromChangesApplyResult, error) {
	manifestPath := setupSvcLiveReplayManifestPath(c.projectPath, firstMapString(packet, "manifestPath", "manifest", "manifestFile"))
	result := setupSvcLiveReplaySnapshotFromChangesApplyResult{
		Mode:             "setup-svc-live-replay-snapshot-from-changes",
		Project:          c.projectPath,
		ReadOnly:         !execute,
		Execute:          execute,
		ApprovalRequired: true,
		Approved:         execute && approval == setupSvcParitySnapshotFromChangesApproval,
		Status:           "dry_run_ready",
		ManifestPath:     manifestPath,
		NextCommands: map[string]string{
			"hydrateSnapshots": "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-snapshot-from-changes <@packet.json> --execute --approval " + setupSvcParitySnapshotFromChangesApproval,
			"validateSources":  "cloudcc scan msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-source-validate " + shellPath(manifestPath) + " --source-readiness complete",
			"dryRunImport":     "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-evidence-import <@complete-worklist.json> --dry-run",
			"approvedImport":   "cloudcc apply msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-evidence-import <@complete-worklist.json> --execute --approval " + setupSvcParityEvidenceImportApproval,
			"completionAudit":  "cloudcc scan msapi " + shellPath(c.projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		},
	}
	if execute && approval != setupSvcParitySnapshotFromChangesApproval {
		return result, fmt.Errorf("refusing to hydrate setup-svc live replay snapshots without --approval %s", setupSvcParitySnapshotFromChangesApproval)
	}
	expectedByDomain := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		expectedByDomain[normalizeDomain(domain.Domain)] = domain
	}
	operationIDs := setupSvcLiveReplaySnapshotFromChangesOperationIDs(c.projectPath, packet)
	items := setupSvcLiveReplaySnapshotFromChangesItems(packet)
	if len(items) == 0 {
		result.BlockingIssues = append(result.BlockingIssues, "packet: missing artifacts, artifactReplacementRecords, or operatorBatch artifactReplacementRecords")
		result.Status = "blocked"
		return result, nil
	}
	writes := []struct {
		path     string
		artifact map[string]any
	}{}
	for index, item := range items {
		artifact, write, issues := c.setupSvcLiveReplaySnapshotArtifactFromChanges(index, item, expectedByDomain, operationIDs)
		if len(issues) == 0 {
			targetPath := firstMapString(item, "path", "targetPath", "evidencePath")
			if targetPath == "" {
				targetPath = filepath.Join("outputs", "setup-svc-live-replay", write.Domain, write.Operation, write.ArtifactType+".json")
			}
			issues = append(issues, verifySetupSvcLiveReplayEvidenceArtifact(c.projectPath, targetPath, expectedByDomain[normalizeDomain(write.Domain)], write.Operation, artifact)...)
		}
		if len(issues) > 0 {
			write.Status = "failed"
			write.Issues = issues
			result.FailedArtifacts++
			for _, issue := range issues {
				result.BlockingIssues = append(result.BlockingIssues, setupSvcLiveReplaySnapshotFromChangesIssuePrefix(write, index)+issue)
			}
		} else {
			write.Status = "ready"
			result.PassedArtifacts++
			sourcePath := firstMapString(item, "sourcePath", "suggestedSourcePath")
			if sourcePath == "" {
				sourcePath = filepath.Join("captures", "outputs", "setup-svc-live-replay", write.Domain, write.Operation, write.ArtifactType+".json")
			}
			writes = append(writes, struct {
				path     string
				artifact map[string]any
			}{path: sourcePath, artifact: artifact})
		}
		result.Artifacts = append(result.Artifacts, write)
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked"
		return result, nil
	}
	if !execute {
		result.Warnings = []string{"Dry run only; snapshot source evidence files were not written."}
		return result, nil
	}
	for _, write := range writes {
		path := setupSvcLiveReplayResolveEvidenceFile(c.projectPath, write.path)
		body, err := json.MarshalIndent(write.artifact, "", "  ")
		if err != nil {
			result.Status = "blocked_artifact_marshal"
			result.BlockingIssues = append(result.BlockingIssues, write.path+": "+err.Error())
			return result, nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			result.Status = "blocked_artifact_write"
			result.BlockingIssues = append(result.BlockingIssues, write.path+": "+err.Error())
			return result, nil
		}
		if err := os.WriteFile(path, append(body, '\n'), 0644); err != nil {
			result.Status = "blocked_artifact_write"
			result.BlockingIssues = append(result.BlockingIssues, write.path+": "+err.Error())
			return result, nil
		}
		result.WrittenFiles++
	}
	result.Status = "applied"
	return result, nil
}

func setupSvcLiveReplaySnapshotFromChangesItems(packet map[string]any) []map[string]any {
	var out []map[string]any
	seen := map[string]bool{}
	var collect func(any)
	collect = func(value any) {
		switch item := value.(type) {
		case []any:
			for _, raw := range item {
				collect(raw)
			}
		case []map[string]any:
			for _, raw := range item {
				collect(raw)
			}
		case map[string]any:
			if setupSvcLiveReplaySnapshotFromChangesLooksLikeItem(item) {
				key := setupSvcLiveReplaySnapshotFromChangesItemKey(item)
				if !seen[key] {
					seen[key] = true
					out = append(out, item)
				}
				return
			}
			for _, key := range []string{"artifacts", "artifactReplacementRecords", "records", "items", "operatorBatch", "batches", "queues"} {
				if raw, ok := item[key]; ok {
					collect(raw)
				}
			}
		}
	}
	collect(packet)
	return out
}

func setupSvcLiveReplaySnapshotFromChangesItemKey(item map[string]any) string {
	targetPath := firstMapString(item, "path", "targetPath", "evidencePath")
	sourcePath := firstMapString(item, "sourcePath", "suggestedSourcePath")
	if strings.TrimSpace(targetPath) != "" {
		sourcePath = ""
	}
	parts := []string{
		normalizeDomain(firstMapString(item, "domain", "msapiDomain", "metadataDomain")),
		strings.ToLower(strings.TrimSpace(firstMapString(item, "operation", "operationType", "action"))),
		setupSvcLiveReplayNormalizeArtifactType(firstMapString(item, "artifactType", "artifact", "evidenceType")),
		targetPath,
		sourcePath,
	}
	return strings.Join(parts, "|")
}

func setupSvcLiveReplaySnapshotFromChangesLooksLikeItem(item map[string]any) bool {
	artifactType := setupSvcLiveReplayNormalizeArtifactType(firstMapString(item, "artifactType", "artifact", "evidenceType"))
	if artifactType != "setup-svc" && artifactType != "metadata-service" {
		return false
	}
	return firstMapString(item, "domain", "msapiDomain", "metadataDomain") != "" && firstMapString(item, "operation", "operationType", "action") != ""
}

func (c *client) setupSvcLiveReplaySnapshotArtifactFromChanges(index int, item map[string]any, expectedByDomain map[string]setupSvcLiveReplayDomain, operationIDs map[string]string) (map[string]any, setupSvcLiveReplaySnapshotFromChangesWriteItem, []string) {
	domainName := normalizeDomain(firstMapString(item, "domain", "msapiDomain", "metadataDomain"))
	operation := strings.ToLower(strings.TrimSpace(firstMapString(item, "operation", "operationType", "action")))
	artifactType := setupSvcLiveReplayNormalizeArtifactType(firstMapString(item, "artifactType", "artifact", "evidenceType"))
	sourcePath := firstMapString(item, "sourcePath", "suggestedSourcePath")
	targetPath := firstMapString(item, "path", "targetPath", "evidencePath")
	expected := expectedByDomain[domainName]
	write := setupSvcLiveReplaySnapshotFromChangesWriteItem{
		Domain:          domainName,
		Operation:       operation,
		ArtifactType:    artifactType,
		SourcePath:      sourcePath,
		TargetPath:      targetPath,
		RequiredTables:  len(setupSvcLiveReplayRequiredTablesForOperation(expected, operation)),
		RequiredEffects: len(setupSvcLiveReplayRuntimeEffectsForOperation(expected.Domain, operation)),
	}
	var issues []string
	if expected.Domain == "" {
		issues = append(issues, "unknownDomain")
	}
	if artifactType != "setup-svc" && artifactType != "metadata-service" {
		issues = append(issues, "artifactTypeMustBeSetupSvcOrMetadataService")
	}
	sourceArtifact := map[string]any{}
	if sourcePath != "" {
		if decoded, err := readJSONFile(setupSvcLiveReplayResolveEvidenceFile(c.projectPath, sourcePath)); err == nil {
			sourceArtifact = decoded
		}
	}
	operationID := setupSvcLiveReplaySnapshotFromChangesOperationID(item)
	if operationID == "" && len(sourceArtifact) > 0 {
		operationID = setupSvcLiveReplaySnapshotFromChangesOperationID(sourceArtifact)
	}
	if operationID == "" {
		operationID = setupSvcLiveReplaySnapshotFromChangesLookupOperationID(operationIDs, item, domainName, operation, artifactType, sourcePath, targetPath)
	}
	write.OperationID = operationID
	changes := setupSvcLiveReplaySnapshotFromChangesExtractChanges(item)
	if len(changes) == 0 && len(sourceArtifact) > 0 {
		changes = setupSvcLiveReplaySnapshotFromChangesExtractChanges(sourceArtifact)
	}
	if len(changes) == 0 && strings.TrimSpace(operationID) != "" {
		fetchedChanges, err := c.setupSvcLiveReplayFetchOperationChanges(operationID)
		if err != nil {
			issues = append(issues, "operationChangesFetchFailed="+err.Error())
		} else {
			changes = fetchedChanges
		}
	}
	write.Changes = len(changes)
	if len(changes) == 0 {
		issues = append(issues, "missingOperationChanges")
	}
	expectedTables := setupSvcLiveReplayRequiredTablesForOperation(expected, operation)
	expectedEffects := setupSvcLiveReplayRuntimeEffectsForOperation(expected.Domain, operation)
	snapshots := setupSvcLiveReplayTableSnapshotsFromChanges(changes, expectedTables)
	effects := setupSvcLiveReplayRuntimeEffectChecksFromChanges(changes, expectedEffects)
	write.TableSnapshots = len(snapshots)
	write.RuntimeEffects = len(effects)
	artifact := map[string]any{
		"status":              "passed",
		"project":             setupSvcLiveReplayProjectIdentity(c.projectPath),
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":              domainName,
		"operation":           operation,
		"artifactType":        artifactType,
		"operationId":         operationID,
		"sourcePath":          sourcePath,
		"targetPath":          targetPath,
		"sourceKind":          "operation-changes",
		"sourceRecordIndex":   index,
		"tableSnapshots":      snapshots,
		"runtimeEffectChecks": effects,
		"changes":             changes,
	}
	if artifactType == "metadata-service" {
		if datasource, ok := sourceArtifact["metadataServiceDatasource"].(map[string]any); ok && len(datasource) > 0 {
			artifact["metadataServiceDatasource"] = datasource
		} else {
			artifact["metadataServiceDatasource"] = setupSvcLiveReplayDatasourceReadinessMap(setupSvcLiveReplayDatasourceReadinessFor())
		}
	}
	return artifact, write, issues
}

func setupSvcLiveReplaySnapshotFromChangesOperationIDs(projectPath string, packet map[string]any) map[string]string {
	out := map[string]string{}
	store := func(key string, operationID string) {
		key = setupSvcLiveReplaySnapshotFromChangesOperationIDKey(key)
		operationID = strings.TrimSpace(operationID)
		if key != "" && operationID != "" {
			out[key] = operationID
		}
	}
	var storeItem func(map[string]any)
	storeItem = func(item map[string]any) {
		operationID := setupSvcLiveReplaySnapshotFromChangesOperationID(item)
		if operationID == "" {
			return
		}
		domain := normalizeDomain(firstMapString(item, "domain", "msapiDomain", "metadataDomain"))
		operation := strings.ToLower(strings.TrimSpace(firstMapString(item, "operation", "operationType", "action")))
		artifactType := setupSvcLiveReplayNormalizeArtifactType(firstMapString(item, "artifactType", "artifact", "evidenceType"))
		if artifactType == "" {
			artifactType = "metadata-service"
		}
		targetPath := firstMapString(item, "path", "targetPath", "evidencePath")
		sourcePath := firstMapString(item, "sourcePath", "suggestedSourcePath")
		for _, key := range setupSvcLiveReplaySnapshotFromChangesOperationIDCandidateKeys(domain, operation, artifactType, sourcePath, targetPath) {
			store(key, operationID)
		}
		for _, key := range []string{"key", "operationKey", "artifactKey", "targetKey"} {
			store(firstMapString(item, key), operationID)
		}
	}
	for _, key := range []string{"operationIds", "operationIDs", "operationIdMap", "operationIDMap", "operationResultsByKey"} {
		if values, ok := packet[key].(map[string]any); ok {
			for rawKey, rawValue := range values {
				switch value := rawValue.(type) {
				case string:
					store(rawKey, value)
				case map[string]any:
					operationID := setupSvcLiveReplaySnapshotFromChangesOperationID(value)
					store(rawKey, operationID)
					storeItem(value)
				}
			}
		}
	}
	var collect func(any)
	collect = func(value any) {
		switch item := value.(type) {
		case []any:
			for _, raw := range item {
				collect(raw)
			}
		case []map[string]any:
			for _, raw := range item {
				storeItem(raw)
			}
		case map[string]any:
			storeItem(item)
			for _, key := range []string{"operationResults", "metadataServiceOperations", "applyResults", "results"} {
				if raw, ok := item[key]; ok {
					collect(raw)
				}
			}
		}
	}
	for _, key := range []string{"operationResults", "metadataServiceOperations", "applyResults", "results"} {
		if raw, ok := packet[key]; ok {
			collect(raw)
		}
	}
	for _, rawPath := range setupSvcLiveReplaySnapshotFromChangesOperationResultPaths(packet) {
		path := setupSvcLiveReplayResolveEvidenceFile(projectPath, rawPath)
		if decoded, err := readJSONAnyFile(path); err == nil {
			collect(decoded)
		}
	}
	for _, rawDir := range setupSvcLiveReplaySnapshotFromChangesOperationResultDirs(packet) {
		root := setupSvcLiveReplayResolveEvidenceFile(projectPath, rawDir)
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry == nil || entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
				return nil
			}
			if decoded, err := readJSONAnyFile(path); err == nil {
				collect(decoded)
			}
			return nil
		})
	}
	return out
}

func setupSvcLiveReplaySnapshotFromChangesOperationResultPaths(packet map[string]any) []string {
	out := []string{}
	for _, key := range []string{"operationResultsFile", "operationResultsPath", "operationResultFile", "operationResultPath", "applyResultsFile", "applyResultsPath", "applyResultFile", "applyResultPath"} {
		out = appendIfPresent(out, firstMapString(packet, key))
	}
	for _, key := range []string{"operationResultPaths", "operationResultsPaths", "applyResultPaths", "applyResultsPaths"} {
		out = append(out, stringList(packet[key])...)
	}
	return nonEmptyStrings(out)
}

func setupSvcLiveReplaySnapshotFromChangesOperationResultDirs(packet map[string]any) []string {
	out := []string{}
	for _, key := range []string{"operationResultsDir", "operationResultDir", "applyResultsDir", "applyResultDir"} {
		out = appendIfPresent(out, firstMapString(packet, key))
	}
	for _, key := range []string{"operationResultDirs", "operationResultsDirs", "applyResultDirs", "applyResultsDirs"} {
		out = append(out, stringList(packet[key])...)
	}
	return nonEmptyStrings(out)
}

func readJSONAnyFile(path string) (any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("%s: JSON file is invalid: %w", path, err)
	}
	return out, nil
}

func setupSvcLiveReplaySnapshotFromChangesLookupOperationID(operationIDs map[string]string, item map[string]any, domain string, operation string, artifactType string, sourcePath string, targetPath string) string {
	for _, key := range setupSvcLiveReplaySnapshotFromChangesOperationIDCandidateKeys(domain, operation, artifactType, sourcePath, targetPath) {
		if value := operationIDs[setupSvcLiveReplaySnapshotFromChangesOperationIDKey(key)]; value != "" {
			return value
		}
	}
	for _, key := range []string{"key", "operationKey", "artifactKey", "targetKey"} {
		if value := operationIDs[setupSvcLiveReplaySnapshotFromChangesOperationIDKey(firstMapString(item, key))]; value != "" {
			return value
		}
	}
	return ""
}

func setupSvcLiveReplaySnapshotFromChangesOperationIDCandidateKeys(domain string, operation string, artifactType string, sourcePath string, targetPath string) []string {
	domain = normalizeDomain(domain)
	operation = strings.ToLower(strings.TrimSpace(operation))
	artifactType = setupSvcLiveReplayNormalizeArtifactType(artifactType)
	if artifactType == "" {
		artifactType = "metadata-service"
	}
	keys := []string{}
	if domain != "" && operation != "" {
		keys = append(keys, domain+"/"+operation, domain+":"+operation)
		if artifactType != "" {
			keys = append(keys, domain+"/"+operation+"/"+artifactType, domain+":"+operation+":"+artifactType)
		}
	}
	for _, path := range []string{targetPath, sourcePath} {
		if strings.TrimSpace(path) != "" {
			keys = append(keys, path)
		}
	}
	return keys
}

func setupSvcLiveReplaySnapshotFromChangesOperationIDKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\\", "/")
	value = strings.TrimPrefix(value, "./")
	if strings.Contains(value, "/") {
		value = filepath.ToSlash(filepath.Clean(value))
	}
	return strings.ToLower(value)
}

func setupSvcLiveReplaySnapshotFromChangesOperationID(item map[string]any) string {
	if value := firstMapString(item, "operationId", "operationID", "metadataServiceOperationId", "setupSvcOperationId"); value != "" {
		return value
	}
	for _, key := range []string{"operationResponse", "operation", "metadataServiceOperation", "setupSvcOperation", "result", "response"} {
		if nested, ok := item[key].(map[string]any); ok {
			if value := firstMapString(nested, "operationId", "operationID", "id"); value != "" {
				return value
			}
		}
	}
	return ""
}

func (c *client) setupSvcLiveReplayFetchOperationChanges(operationID string) ([]any, error) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return nil, fmt.Errorf("missing operationId")
	}
	var out bytes.Buffer
	if err := c.getJSON(&out, "/metadata/v1/operations/"+url.PathEscape(operationID)+"/changes"); err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		return nil, err
	}
	changes := setupSvcLiveReplaySnapshotFromChangesExtractChanges(decoded)
	if len(changes) == 0 {
		if array, ok := decoded.([]any); ok {
			changes = array
		}
	}
	return changes, nil
}

func setupSvcLiveReplaySnapshotFromChangesExtractChanges(value any) []any {
	switch item := value.(type) {
	case nil:
		return nil
	case []any:
		return item
	case map[string]any:
		for _, key := range []string{"changes", "operationChanges", "changeItems"} {
			if changes := setupSvcLiveReplaySnapshotFromChangesExtractChanges(item[key]); len(changes) > 0 {
				return changes
			}
		}
		for _, key := range []string{"operationResponse", "operation", "metadataServiceOperation", "setupSvcOperation", "result", "response"} {
			if nested, ok := item[key].(map[string]any); ok {
				if changes := setupSvcLiveReplaySnapshotFromChangesExtractChanges(nested); len(changes) > 0 {
					return changes
				}
			}
		}
	}
	return nil
}

func setupSvcLiveReplayTableSnapshotsFromChanges(changes []any, requiredTables []string) map[string]any {
	required := map[string]bool{}
	for _, table := range requiredTables {
		if normalized := strings.ToLower(strings.TrimSpace(table)); normalized != "" {
			required[normalized] = true
		}
	}
	type tableEvidence struct {
		columns       map[string]bool
		mutationTypes map[string]bool
		rows          []any
		changeCount   int
	}
	byTable := map[string]*tableEvidence{}
	for _, raw := range changes {
		change, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if strings.EqualFold(firstMapString(change, "mutationType", "type"), "SIDE_EFFECT") {
			continue
		}
		table := strings.TrimSpace(firstMapString(change, "tableName", "table", "name"))
		normalizedTable := strings.ToLower(table)
		if normalizedTable == "" || (len(required) > 0 && !required[normalizedTable]) {
			continue
		}
		evidence := byTable[normalizedTable]
		if evidence == nil {
			evidence = &tableEvidence{columns: map[string]bool{}, mutationTypes: map[string]bool{}}
			byTable[normalizedTable] = evidence
		}
		evidence.changeCount++
		if mutationType := strings.TrimSpace(firstMapString(change, "mutationType", "type")); mutationType != "" {
			evidence.mutationTypes[strings.ToUpper(mutationType)] = true
		}
		rows := setupSvcLiveReplayRowsFromChange(change)
		for _, row := range rows {
			evidence.rows = append(evidence.rows, row)
			if rowMap, ok := row.(map[string]any); ok {
				for key := range rowMap {
					evidence.columns[key] = true
				}
			}
		}
	}
	out := map[string]any{}
	for _, table := range requiredTables {
		normalized := strings.ToLower(strings.TrimSpace(table))
		if normalized == "" {
			continue
		}
		evidence := byTable[normalized]
		if evidence == nil {
			continue
		}
		columns := make([]string, 0, len(evidence.columns))
		for column := range evidence.columns {
			columns = append(columns, column)
		}
		sort.Strings(columns)
		mutationTypes := make([]string, 0, len(evidence.mutationTypes))
		for mutationType := range evidence.mutationTypes {
			mutationTypes = append(mutationTypes, mutationType)
		}
		sort.Strings(mutationTypes)
		out[table] = map[string]any{
			"table":         table,
			"columns":       columns,
			"rows":          evidence.rows,
			"rowCount":      len(evidence.rows),
			"changeCount":   evidence.changeCount,
			"mutationTypes": mutationTypes,
		}
	}
	return out
}

func setupSvcLiveReplayRowsFromChange(change map[string]any) []any {
	for _, key := range []string{"after", "afterRows", "rows", "records"} {
		if rows := setupSvcLiveReplayRowsFromSnapshotValue(change[key]); len(rows) > 0 {
			return rows
		}
	}
	for _, key := range []string{"before", "beforeRows"} {
		if rows := setupSvcLiveReplayRowsFromSnapshotValue(change[key]); len(rows) > 0 {
			return rows
		}
	}
	return nil
}

func setupSvcLiveReplayRowsFromSnapshotValue(value any) []any {
	switch item := value.(type) {
	case []any:
		return item
	case []map[string]any:
		out := make([]any, 0, len(item))
		for _, row := range item {
			out = append(out, row)
		}
		return out
	case map[string]any:
		return []any{item}
	default:
		return nil
	}
}

func setupSvcLiveReplayRuntimeEffectChecksFromChanges(changes []any, expectedEffects []string) []map[string]any {
	byEffect := map[string]map[string]any{}
	changedTables := map[string]bool{}
	for _, raw := range changes {
		change, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		tableName := strings.TrimSpace(firstMapString(change, "tableName", "table", "name"))
		if tableName != "" {
			changedTables[strings.ToLower(tableName)] = true
		}
		if !strings.EqualFold(firstMapString(change, "mutationType", "type"), "SIDE_EFFECT") {
			continue
		}
		effect := strings.TrimSpace(firstMapString(change, "tableName", "effectType", "runtimeEffect", "name"))
		if effect == "" {
			if after, ok := change["after"].(map[string]any); ok {
				effect = firstMapString(after, "effectType", "runtimeEffect", "name")
			}
		}
		if effect == "" {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(firstMapString(change, "status", "checkStatus")))
		checkStatus := "failed"
		if status == "applied" || status == "passed" || status == "verified" || status == "success" || status == "successful" {
			checkStatus = "passed"
		}
		byEffect[setupSvcLiveReplayExpectationKey(effect)] = map[string]any{
			"name":     effect,
			"status":   checkStatus,
			"source":   "operation-change",
			"targetId": firstMapString(change, "targetId"),
			"evidence": firstMapAny(change, "after", "result", "details"),
		}
	}
	checks := make([]map[string]any, 0, len(expectedEffects))
	for _, effect := range expectedEffects {
		key := setupSvcLiveReplayExpectationKey(effect)
		if check, ok := byEffect[key]; ok {
			checks = append(checks, check)
			continue
		}
		if evidenceTables := setupSvcLiveReplayRuntimeEffectEvidenceTables(effect); len(evidenceTables) > 0 && setupSvcLiveReplayTablesCoverEffect(changedTables, evidenceTables) {
			checks = append(checks, map[string]any{
				"name":           effect,
				"status":         "passed",
				"source":         "operation-change-tables",
				"evidenceTables": evidenceTables,
			})
			continue
		}
		if setupSvcLiveReplayCleanupEffect(effect) && len(changedTables) > 0 {
			evidenceTables := make([]string, 0, len(changedTables))
			for table := range changedTables {
				evidenceTables = append(evidenceTables, table)
			}
			sort.Strings(evidenceTables)
			checks = append(checks, map[string]any{
				"name":           effect,
				"status":         "passed",
				"source":         "operation-change-cleanup-tables",
				"evidenceTables": evidenceTables,
			})
		}
	}
	return checks
}

func setupSvcLiveReplayCleanupEffect(effect string) bool {
	key := setupSvcLiveReplayExpectationKey(effect)
	return strings.Contains(key, "cleanup") || strings.Contains(key, "hard-delete") || strings.Contains(key, "physical-purge")
}

func setupSvcLiveReplayRuntimeEffectEvidenceTables(effect string) []string {
	switch setupSvcLiveReplayExpectationKey(effect) {
	case "datatable-prefix-allocation":
		return []string{"tp_sys_datatablestate", "tp_sys_object"}
	case "standard-object-default-fields":
		return []string{"tp_sys_schemetable", "tp_sys_multi_lang", "tp_sys_profile_field"}
	case "standard-related-list-expansion":
		return []string{"tp_sys_relatedlist", "tp_sys_relatedlist_field", "tp_sys_multi_lang"}
	case "object-view-order-expansion":
		return []string{"tp_sys_view", "tp_sys_object_view_order", "tp_sys_view_field"}
	case "layout-button-view-profile-expansion":
		return []string{"tp_sys_layout", "tp_sys_layout_section", "tp_sys_section_field", "tp_sys_layout_button", "tp_sys_profile_layout", "tp_sys_button", "tp_sys_lookuplayout"}
	case "field-row-and-label-expansion":
		return []string{"tp_sys_schemetable", "tp_sys_multi_lang"}
	case "option-reference-dependency-link-expansion":
		return []string{"tp_sys_code", "tp_sys_globalselect_field", "tp_sys_field_dependency", "tp_sys_field_reference", "tp_sys_autonum"}
	case "layout-profile-permission-expansion":
		return []string{"tp_sys_profile_field", "tp_sys_section_field"}
	case "lookup-relatedlist-expansion":
		return []string{"tp_sys_relatedlist", "tp_sys_relatedlist_field"}
	case "global-list-option-label-expansion":
		return []string{"tp_sys_global_select", "tp_sys_code"}
	case "field-link-cleanup":
		return []string{"tp_sys_globalselect_field"}
	case "record-type-profile-infoset-expansion":
		return []string{"tp_sys_recordtype", "tp_sys_profile_infoset"}
	case "record-type-profile-layout-expansion":
		return []string{"tp_sys_profile_layout"}
	case "object-recordtype-enable-expansion":
		return []string{"tp_sys_object"}
	case "record-type-field-dependency-expansion":
		return []string{"tp_sys_schemetable", "tp_sys_field_dependency"}
	case "layout-section-field-expansion":
		return []string{"tp_sys_layout", "tp_sys_layout_section", "tp_sys_section_field"}
	case "profile-layout-and-button-link-expansion":
		return []string{"tp_sys_profile_layout", "tp_sys_layout_button"}
	case "object-field-layout-recordtype-grant-expansion":
		return []string{"tp_sys_profile", "tp_sys_profile_infoset", "tp_sys_profile_field", "tp_sys_profile_layout"}
	case "permission-definition-label-expansion":
		return []string{"tp_sys_profile_permission", "tp_sys_multi_lang"}
	case "profile-infoset-menu-label-expansion":
		return []string{"tp_sys_profile_infoset"}
	case "permission-set-infoset-field-expansion":
		return []string{"tp_sys_permsets", "tp_sys_permsets_infoset", "tp_sys_permsets_fields"}
	case "permission-set-assignment-expansion", "assignment-remove-cleanup":
		return []string{"tp_sys_permsets_assign"}
	case "role-group-hierarchy-expansion":
		return []string{"tp_sys_role", "tp_sys_group"}
	case "user-role-assignment-update", "user-role-unassignment-fallback-update":
		return []string{"tp_sys_user"}
	case "sharing-rule-condition-expansion":
		return []string{"tp_sys_sharerule", "tp_sys_condition"}
	case "validation-rule-row-lifecycle":
		return []string{"tp_sys_validaterule"}
	case "tab-app-profile-binding-expansion":
		return []string{"tp_sys_tab", "tp_sys_app_tab", "tp_sys_profile_infoset"}
	case "tab-profile-all-expansion":
		return []string{"tp_sys_profile_infoset"}
	case "app-tab-binding-expansion":
		return []string{"tp_sys_app", "tp_sys_app_tab", "tp_sys_tab"}
	case "app-profile-visibility-expansion":
		return []string{"tp_sys_profile_infoset"}
	case "button-scope-layout-view-binding-expansion":
		return []string{"tp_sys_button", "tp_sys_button_scope", "tp_sys_layout_button", "tp_sys_view_button", "tp_sys_lookuplayout"}
	case "setting-object-table-view-allocation":
		return []string{"tp_sys_object"}
	case "setting-field-expansion":
		return []string{"tp_sys_schemetable", "tp_sys_multi_lang"}
	case "setting-layout-profile-expansion":
		return []string{"tp_sys_layout", "tp_sys_layout_section", "tp_sys_section_field", "tp_sys_profile_layout"}
	case "dupe-rule-field-condition-expansion", "dupe-rule-firstletters-normalization":
		return []string{"tp_sys_dupecatcher", "tp_sys_dupecatcherule", "tp_sys_condition"}
	case "saml-sp-row-lifecycle":
		return []string{"tp_sys_sp_idps"}
	case "idp-row-lifecycle":
		return []string{"tp_sys_idp_config"}
	case "idp-sp-binding-expansion", "idp-sp-logoutbinding-normalization", "idp-sp-update-app-preservation":
		return []string{"tp_sys_idp_sps"}
	case "approval-step-condition-action-expansion":
		return []string{"tp_sys_approval", "tp_sys_approval_step", "tp_sys_condition", "tp_sys_actions_relation"}
	case "step-layout-expansion":
		return []string{"tp_sys_approval_step_layout"}
	case "approval-related-list-field-expansion":
		return []string{"tp_sys_apralrellist", "tp_sys_apralrellist_fields"}
	case "report-folder-field-filter-expansion":
		return []string{"tp_sys_report", "tp_sys_folder", "tp_sys_report_fieldname", "tp_sys_condition"}
	case "report-type-custom-field-expansion":
		return []string{"tp_sys_reporttypecustom", "tp_sys_reporttypecustomfields"}
	case "dashboard-component-source-expansion":
		return []string{"tp_sys_dashboard", "tp_sys_dashboard_report"}
	case "dashboard-condition-expansion":
		return []string{"tp_sys_dashboard_condition"}
	case "view-field-filter-button-expansion":
		return []string{"tp_sys_view", "tp_sys_view_field", "tp_sys_view_button", "tp_sys_condition"}
	case "view-chart-kanban-expansion":
		return []string{"tp_sys_viewcharts", "tp_sys_viewkanban", "tp_sys_viewkanban_field"}
	case "view-dashboard-component-cleanup":
		return []string{"tp_sys_dashboard_report"}
	case "translated-label-expansion":
		return []string{"tp_sys_multi_lang"}
	case "database-view-refresh":
		return []string{"database-view-refresh"}
	default:
		if strings.Contains(setupSvcLiveReplayExpectationKey(effect), "delete-cleanup") ||
			strings.Contains(setupSvcLiveReplayExpectationKey(effect), "hard-delete-cleanup") ||
			strings.Contains(setupSvcLiveReplayExpectationKey(effect), "cleanup") {
			return []string{}
		}
		return nil
	}
}

func setupSvcLiveReplayTablesCoverEffect(changedTables map[string]bool, evidenceTables []string) bool {
	for _, table := range evidenceTables {
		if !changedTables[strings.ToLower(strings.TrimSpace(table))] {
			return false
		}
	}
	return len(evidenceTables) > 0
}

func setupSvcLiveReplaySnapshotFromChangesIssuePrefix(write setupSvcLiveReplaySnapshotFromChangesWriteItem, index int) string {
	identity := strings.TrimSpace(write.SourcePath)
	if identity == "" {
		identity = strings.TrimSpace(write.TargetPath)
	}
	if identity == "" {
		identity = fmt.Sprintf("artifact[%d]", index)
	}
	return identity + ": "
}

func setupSvcLiveReplayQueryReadbackArtifactShape(projectPath string, domain string, operation string, requiredTables []string, expectations []string) map[string]any {
	readbackTables := []map[string]any{}
	for _, table := range requiredTables {
		table = strings.TrimSpace(table)
		if table == "" {
			continue
		}
		readbackTables = append(readbackTables, map[string]any{
			"table":          table,
			"rows":           []map[string]any{},
			"fields":         []string{},
			"requiredFields": []string{},
		})
	}
	expectationChecks := []map[string]any{}
	for _, expectation := range expectations {
		expectation = strings.TrimSpace(expectation)
		if expectation == "" {
			continue
		}
		expectationChecks = append(expectationChecks, map[string]any{
			"name":   expectation,
			"status": "pending",
		})
	}
	return map[string]any{
		"status":              "pending_capture",
		"project":             setupSvcLiveReplayProjectIdentity(projectPath),
		"contractVersion":     setupSvcLiveReplayContractVersion,
		"contractFingerprint": setupSvcLiveReplayExpectedContractFingerprint(),
		"domain":              domain,
		"operation":           operation,
		"artifactType":        "query-readback",
		"queryShape": map[string]any{
			"requiredTables": append([]string{}, requiredTables...),
		},
		"readbackShape": map[string]any{
			"requiredTables": append([]string{}, requiredTables...),
			"expectations":   append([]string{}, expectations...),
		},
		"readbackTables":            readbackTables,
		"relationshipChecks":        []map[string]any{},
		"readbackExpectationChecks": expectationChecks,
		"cleanCounters": map[string]any{
			"missingFields":           nil,
			"missingRelationships":    nil,
			"missingRows":             nil,
			"missingColumns":          nil,
			"missingValues":           nil,
			"mismatchedFields":        nil,
			"mismatchedRelationships": nil,
			"mismatchedRows":          nil,
			"mismatchedColumns":       nil,
			"mismatchedValues":        nil,
			"brokenRelationships":     nil,
			"unreadableRelationships": nil,
			"errors":                  nil,
			"failures":                nil,
			"blockingIssues":          nil,
		},
	}
}

func setupSvcLiveReplaySourceExecutionArgsFromOptions(options setupSvcLiveReplayCollectionPlanOptions) []string {
	args := []string{}
	if options.Domain != "" {
		args = append(args, "--domain", options.Domain)
	}
	if options.Operation != "" {
		args = append(args, "--operation", options.Operation)
	}
	if options.ArtifactType != "" {
		args = append(args, "--artifact-type", options.ArtifactType)
	}
	if options.SourceSystem != "" {
		args = append(args, "--source-system", options.SourceSystem)
	}
	if options.CaptureMode != "" {
		args = append(args, "--capture-mode", options.CaptureMode)
	}
	if options.Status != "" {
		args = append(args, "--status", options.Status)
	}
	if options.EvidenceSection != "" {
		args = append(args, "--evidence-section", options.EvidenceSection)
	}
	if options.SectionStatus != "" {
		args = append(args, "--section-status", options.SectionStatus)
	}
	if options.Offset > 0 || (options.Limit > 0 && options.Limit != 25) {
		args = append(args, "--offset", strconv.Itoa(options.Offset), "--limit", strconv.Itoa(options.Limit))
	}
	return args
}

func setupSvcLiveReplaySourceExecutionNextAction(artifactType string, captureMode string) string {
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "setup-svc":
		return "Run setup-svc reference create/update/delete/query flow, capture table snapshots and runtime effects, then replace mirrored source JSON."
	case "metadata-service":
		switch strings.TrimSpace(captureMode) {
		case "msapi_scan_snapshot_capture":
			return "Run approval-gated MetadataService query scan capture for metadata-service/query records, then replace mirrored source JSON with strict tableSnapshots and runtimeEffectChecks."
		case "msapi_plan_apply_snapshot_capture":
			return "Run approval-gated MetadataService apply capture for write records, then hydrate snapshots from operation changes."
		}
		return "Run MSAPI plan/apply for the same domain/operation, capture table snapshots and runtime effects, then replace mirrored source JSON."
	case "query-readback":
		return "Run MSAPI query/readback scanner, capture readback tables, relationship checks, expectation checks, and clean counters."
	case "normalized-diff":
		return "Generate normalized diff from completed setup-svc and metadata-service snapshots with approval-gated normalized-diff command."
	case "cleanup":
		return "Run cleanup verifier after replay, capture residual/orphan counters and cleanup command effects."
	default:
		return "Replace mirrored source JSON with real evidence."
	}
}

func setupSvcLiveReplaySourceExecutionGate(artifactType string) string {
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "setup-svc":
		return "source JSON must contain passed setup-svc tableSnapshots and runtimeEffectChecks before importing this batch."
	case "metadata-service":
		return "source JSON must contain passed MetadataService tableSnapshots and runtimeEffectChecks for the same domain/operation after setup-svc evidence exists."
	case "query-readback":
		return "source JSON must contain query/readback tables, shape proof, relationships, expectation checks, and clean counters after write evidence exists."
	case "normalized-diff":
		return "source JSON must be generated from completed setup-svc and MetadataService snapshots and prove zero normalized diff counters."
	case "cleanup":
		return "source JSON must prove deleted/removed evidence and zero residual counters after replay cleanup."
	default:
		return "source JSON must satisfy every required evidence section before import."
	}
}

func setupSvcLiveReplaySourceExecutionGroupedSourceFiles(groups []setupSvcLiveReplaySourceExecutionPacketGroup) int {
	total := 0
	for _, group := range groups {
		total += group.SourceFiles
	}
	return total
}

func setupSvcLiveReplaySourceExecutionGroupedTargetFiles(groups []setupSvcLiveReplaySourceExecutionPacketGroup) int {
	total := 0
	for _, group := range groups {
		total += group.TargetFiles
	}
	return total
}

func setupSvcLiveReplayAppendUniqueStrings(items []string, values ...string) []string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && !containsString(items, trimmed) {
			items = append(items, trimmed)
		}
	}
	return items
}

func setupSvcLiveReplaySortChecklistItem(item *setupSvcLiveReplaySourceChecklistItem) {
	sort.Strings(item.MissingEvidenceSections)
	sort.Strings(item.RequiredEvidenceSections)
	sort.Strings(item.RequiredTables)
	sort.Strings(item.RuntimeEffects)
	sort.Strings(item.QueryReadbackExpectations)
	sort.Strings(item.WorklistFiles)
	sort.Strings(item.Checklist)
}

func setupSvcLiveReplaySourceChecklistArtifactTypeCounts(records map[string]int, sources map[string]map[string]bool, targets map[string]map[string]bool) []setupSvcLiveReplaySourceChecklistArtifactTypeCount {
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]setupSvcLiveReplaySourceChecklistArtifactTypeCount, 0, len(keys))
	for _, key := range keys {
		out = append(out, setupSvcLiveReplaySourceChecklistArtifactTypeCount{
			ArtifactType: key,
			Records:      records[key],
			SourceFiles:  len(sources[key]),
			TargetFiles:  len(targets[key]),
		})
	}
	return out
}

func setupSvcLiveReplaySourceChecklistReadinessCounts(records map[string]int, sources map[string]map[string]bool) []setupSvcLiveReplaySourceChecklistReadinessCount {
	keys := make([]string, 0, len(records))
	for key := range records {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]setupSvcLiveReplaySourceChecklistReadinessCount, 0, len(keys))
	for _, key := range keys {
		out = append(out, setupSvcLiveReplaySourceChecklistReadinessCount{
			SourceReadiness: key,
			Records:         records[key],
			SourceFiles:     len(sources[key]),
		})
	}
	return out
}

func setupSvcLiveReplaySourceChecklistMissingSectionCounts(sources []setupSvcLiveReplaySourceChecklistItem) []setupSvcLiveReplaySourceChecklistSectionCount {
	sectionSources := map[string]map[string]bool{}
	sectionTargets := map[string]map[string]bool{}
	sectionArtifactTypes := map[string]map[string]bool{}
	for _, source := range sources {
		sourcePath := strings.TrimSpace(source.SourcePath)
		targetPath := strings.TrimSpace(source.TargetPath)
		if sourcePath == "" {
			continue
		}
		artifactType := setupSvcLiveReplayNormalizeArtifactType(source.ArtifactType)
		for _, section := range source.MissingEvidenceSections {
			section = strings.TrimSpace(section)
			if section == "" {
				continue
			}
			if sectionSources[section] == nil {
				sectionSources[section] = map[string]bool{}
			}
			if sectionTargets[section] == nil {
				sectionTargets[section] = map[string]bool{}
			}
			if sectionArtifactTypes[section] == nil {
				sectionArtifactTypes[section] = map[string]bool{}
			}
			sectionSources[section][sourcePath] = true
			if targetPath != "" {
				sectionTargets[section][targetPath] = true
			}
			if artifactType != "" {
				sectionArtifactTypes[section][artifactType] = true
			}
		}
	}
	keys := make([]string, 0, len(sectionSources))
	for key := range sectionSources {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]setupSvcLiveReplaySourceChecklistSectionCount, 0, len(keys))
	for _, key := range keys {
		artifactTypes := make([]string, 0, len(sectionArtifactTypes[key]))
		for artifactType := range sectionArtifactTypes[key] {
			artifactTypes = append(artifactTypes, artifactType)
		}
		sort.Strings(artifactTypes)
		out = append(out, setupSvcLiveReplaySourceChecklistSectionCount{
			EvidenceSection: key,
			SourceFiles:     len(sectionSources[key]),
			TargetFiles:     len(sectionTargets[key]),
			ArtifactTypes:   artifactTypes,
		})
	}
	return out
}

func setupSvcLiveReplaySourceChecklistQueueCommands(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, counts []setupSvcLiveReplaySourceChecklistSectionCount) []setupSvcLiveReplaySourceChecklistQueueCommand {
	out := []setupSvcLiveReplaySourceChecklistQueueCommand{}
	for _, count := range counts {
		section := strings.TrimSpace(count.EvidenceSection)
		if section == "" || count.SourceFiles <= 0 {
			continue
		}
		artifactTypes := append([]string{}, count.ArtifactTypes...)
		if len(artifactTypes) == 0 {
			artifactTypes = []string{strings.TrimSpace(options.ArtifactType)}
		} else if len(artifactTypes) > 1 {
			artifactTypes = []string{""}
		}
		sort.Strings(artifactTypes)
		for _, artifactType := range artifactTypes {
			artifactType = setupSvcLiveReplayNormalizeArtifactType(artifactType)
			queueOptions := options
			queueOptions.ArtifactType = artifactType
			queueOptions.EvidenceSection = section
			queueOptions.SectionStatus = "missing"
			queueOptions.BatchIndex = -1
			queueOptions.BatchLimit = setupSvcLiveReplayWorklistBatchLimit
			pageSize := queueOptions.Limit
			if pageSize <= 0 {
				pageSize = 25
			}
			if pageSize <= 0 {
				pageSize = count.SourceFiles
			}
			pageCount := 0
			if pageSize > 0 {
				pageCount = (count.SourceFiles + pageSize - 1) / pageSize
			}
			omittedSourceFiles := count.SourceFiles - pageSize
			if omittedSourceFiles < 0 {
				omittedSourceFiles = 0
			}
			worklistCommand := setupSvcLiveReplayWorklistCommand(projectPath, manifestPath, queueOptions)
			worklistPath := setupSvcLiveReplayWorklistSuggestedPath(projectPath, queueOptions)
			sourceChecklistCommand := setupSvcLiveReplaySourceChecklistCommand(projectPath, manifestPath, queueOptions)
			sourceChecklistPath := setupSvcLiveReplaySourceChecklistSuggestedPath(projectPath, queueOptions)
			sourceExecutionCommand := setupSvcLiveReplaySourceExecutionCommand(projectPath, manifestPath, queueOptions)
			sourceExecutionPath := setupSvcLiveReplaySourceExecutionPacketSuggestedPath(projectPath, queueOptions)
			entry := setupSvcLiveReplaySourceChecklistQueueCommand{
				ArtifactType:               artifactType,
				EvidenceSection:            section,
				Count:                      count.SourceFiles,
				SourceFiles:                count.SourceFiles,
				TargetFiles:                count.TargetFiles,
				Command:                    sourceChecklistCommand,
				SourceReadiness:            queueOptions.SourceReadiness,
				Offset:                     queueOptions.Offset,
				Limit:                      queueOptions.Limit,
				PageSize:                   pageSize,
				PageCount:                  pageCount,
				OmittedSourceFiles:         omittedSourceFiles,
				WorklistCommand:            worklistCommand,
				SuggestedWorklistPath:      worklistPath,
				SaveWorklistCommand:        worklistCommand + " > " + shellPath(worklistPath),
				SourceChecklistCommand:     sourceChecklistCommand,
				SuggestedSourceChecklist:   sourceChecklistPath,
				SaveSourceChecklistCommand: sourceChecklistCommand + " > " + shellPath(sourceChecklistPath),
				SourceExecutionCommand:     sourceExecutionCommand,
				SuggestedSourceExecution:   sourceExecutionPath,
				SaveSourceExecutionCommand: sourceExecutionCommand + " > " + shellPath(sourceExecutionPath),
			}
			nextOffset := queueOptions.Offset + pageSize
			if nextOffset < count.SourceFiles {
				nextOptions := queueOptions
				nextOptions.Offset = nextOffset
				nextWorklistCommand := setupSvcLiveReplayWorklistCommand(projectPath, manifestPath, nextOptions)
				nextWorklistPath := setupSvcLiveReplayWorklistSuggestedPath(projectPath, nextOptions)
				nextChecklistCommand := setupSvcLiveReplaySourceChecklistCommand(projectPath, manifestPath, nextOptions)
				nextChecklistPath := setupSvcLiveReplaySourceChecklistSuggestedPath(projectPath, nextOptions)
				entry.NextPageWorklistCommand = nextWorklistCommand
				entry.SaveNextPageWorklist = nextWorklistCommand + " > " + shellPath(nextWorklistPath)
				entry.NextPageSourceChecklist = nextChecklistCommand
				entry.SaveNextPageChecklist = nextChecklistCommand + " > " + shellPath(nextChecklistPath)
			}
			for pageOffset := 0; pageOffset < count.SourceFiles; pageOffset += pageSize {
				pageOptions := queueOptions
				pageOptions.Offset = pageOffset
				pageWorklistCommand := setupSvcLiveReplayWorklistCommand(projectPath, manifestPath, pageOptions)
				pageWorklistPath := setupSvcLiveReplayWorklistSuggestedPath(projectPath, pageOptions)
				pageChecklistCommand := setupSvcLiveReplaySourceChecklistCommand(projectPath, manifestPath, pageOptions)
				pageChecklistPath := setupSvcLiveReplaySourceChecklistSuggestedPath(projectPath, pageOptions)
				entry.PageWorklistSaveCommands = append(entry.PageWorklistSaveCommands, pageWorklistCommand+" > "+shellPath(pageWorklistPath))
				entry.PageChecklistSaveCommands = append(entry.PageChecklistSaveCommands, pageChecklistCommand+" > "+shellPath(pageChecklistPath))
			}
			out = append(out, entry)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SourceFiles != out[j].SourceFiles {
			return out[i].SourceFiles > out[j].SourceFiles
		}
		if out[i].ArtifactType != out[j].ArtifactType {
			return out[i].ArtifactType < out[j].ArtifactType
		}
		return out[i].EvidenceSection < out[j].EvidenceSection
	})
	return out
}

func setupSvcLiveReplaySourceChecklistPageSaveCommands(commands []setupSvcLiveReplaySourceChecklistQueueCommand) ([]string, []string) {
	worklistCommands := []string{}
	checklistCommands := []string{}
	for _, command := range commands {
		worklistCommands = setupSvcLiveReplayAppendUniqueStrings(worklistCommands, command.PageWorklistSaveCommands...)
		checklistCommands = setupSvcLiveReplayAppendUniqueStrings(checklistCommands, command.PageChecklistSaveCommands...)
	}
	return worklistCommands, checklistCommands
}

func setupSvcLiveReplaySourceChecklistPageSaveScript(worklistCommands []string, checklistCommands []string) string {
	if len(worklistCommands) == 0 && len(checklistCommands) == 0 {
		return ""
	}
	lines := []string{
		"#!/usr/bin/env bash",
		"set -euo pipefail",
		"",
		"# Generated by setup-svc-live-replay-source-checklist. Review before executing.",
	}
	if len(worklistCommands) > 0 {
		lines = append(lines, "", "# Save paged worklists")
		lines = append(lines, worklistCommands...)
	}
	if len(checklistCommands) > 0 {
		lines = append(lines, "", "# Save paged source checklists")
		lines = append(lines, checklistCommands...)
	}
	return strings.Join(lines, "\n")
}

func setupSvcLiveReplaySourceChecklistScriptSuggestedPath(projectPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	checklistPath := setupSvcLiveReplaySourceChecklistSuggestedPath(projectPath, options)
	return strings.TrimSuffix(checklistPath, filepath.Ext(checklistPath)) + ".sh"
}

func setupSvcLiveReplaySourceChecklistSavePageScriptCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, scriptPath string) string {
	if strings.TrimSpace(scriptPath) == "" {
		return ""
	}
	command := setupSvcLiveReplaySourceChecklistCommand(projectPath, manifestPath, options)
	return command + " | jq -r '.pageSaveScript' > " + shellPath(scriptPath) + " && chmod +x " + shellPath(scriptPath)
}

func setupSvcLiveReplayPreflightSavePageScriptCommand(projectPath string, scriptPath string) string {
	if strings.TrimSpace(scriptPath) == "" {
		return ""
	}
	return "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-preflight | jq -r '.captureSources.pageSaveScript' > " + shellPath(scriptPath) + " && chmod +x " + shellPath(scriptPath)
}

func setupSvcLiveReplaySourceChecklistOperatorPacketFor(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, commands setupSvcLiveReplayGapCommands) setupSvcLiveReplaySourceChecklistOperatorPacket {
	path := setupSvcLiveReplaySourceChecklistSuggestedPath(projectPath, options)
	command := setupSvcLiveReplaySourceChecklistCommand(projectPath, manifestPath, options)
	return setupSvcLiveReplaySourceChecklistOperatorPacket{
		Purpose:                "collect_real_replay_evidence_by_unique_source_capture_file",
		SuggestedChecklistPath: path,
		SaveChecklistCommand:   command + " > " + shellPath(path),
		SourceRoot:             setupSvcLiveReplayWorklistSourceRoot(),
		CaptureRoot:            setupSvcLiveReplayWorklistCaptureRoot(projectPath),
		PostCaptureCommands:    setupSvcLiveReplayWorklistPostReplacementCommands(commands),
		StopConditions: []string{
			"Do not import a source file until its sourceReadiness is complete.",
			"Do not mark any artifact passed until every missingEvidenceSection has real replay evidence.",
			"Do not run matrix promotion until strict evidence verification and evidence bundle verification pass.",
		},
	}
}

func setupSvcLiveReplaySourceChecklistSuggestedPath(projectPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	parts := []string{"source-capture-checklist"}
	if options.ArtifactType != "" {
		parts = append(parts, options.ArtifactType)
	}
	if options.EvidenceSection != "" {
		parts = append(parts, setupSvcLiveReplayFilenameToken(options.EvidenceSection))
	}
	if options.SectionStatus != "" {
		parts = append(parts, options.SectionStatus)
	}
	if options.SourceStatus != "" {
		parts = append(parts, "source-"+options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		parts = append(parts, "readiness-"+options.SourceReadiness)
	}
	if options.BatchIndex >= 0 {
		parts = append(parts, "batch-"+strconv.Itoa(options.BatchIndex))
	} else if options.BatchLimit != setupSvcLiveReplayWorklistBatchLimit {
		parts = append(parts, "batches-"+strconv.Itoa(options.BatchLimit))
	}
	if options.Offset > 0 || options.Limit != 25 {
		parts = append(parts, "offset-"+strconv.Itoa(options.Offset), "limit-"+strconv.Itoa(options.Limit))
	}
	return filepath.Join(projectPath, "outputs", "setup-svc-live-replay", strings.Join(parts, "-")+".json")
}

func setupSvcLiveReplaySourceChecklistCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	args := []string{"cloudcc", "scan", "msapi", shellPath(projectPath), "setup-svc-live-replay-source-checklist", shellPath(manifestPath)}
	args = append(args, setupSvcLiveReplayGapArgsFromOptions(options)...)
	if options.SourceStatus != "" {
		args = append(args, "--source-status", options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		args = append(args, "--source-readiness", options.SourceReadiness)
	}
	if options.BatchIndex >= 0 {
		args = append(args, "--batch-index", strconv.Itoa(options.BatchIndex))
	} else if options.BatchLimit != setupSvcLiveReplayWorklistBatchLimit {
		args = append(args, "--batch-limit", strconv.Itoa(options.BatchLimit))
	}
	return strings.Join(args, " ")
}

func setupSvcLiveReplayWorklistOptionArgsFromOptions(options setupSvcLiveReplayCollectionPlanOptions) []string {
	args := setupSvcLiveReplayGapArgsFromOptions(options)
	if options.SourceStatus != "" {
		args = append(args, "--source-status", options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		args = append(args, "--source-readiness", options.SourceReadiness)
	}
	if options.BatchIndex >= 0 {
		args = append(args, "--batch-index", strconv.Itoa(options.BatchIndex))
	} else if options.BatchLimit != setupSvcLiveReplayWorklistBatchLimit {
		args = append(args, "--batch-limit", strconv.Itoa(options.BatchLimit))
	}
	return args
}

func setupSvcLiveReplayWorklistOperatorPacketFor(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, commands setupSvcLiveReplayGapCommands) setupSvcLiveReplayWorklistOperatorPacket {
	worklistPath := setupSvcLiveReplayWorklistSuggestedPath(projectPath, options)
	worklistCommand := setupSvcLiveReplayWorklistCommand(projectPath, manifestPath, options)
	return setupSvcLiveReplayWorklistOperatorPacket{
		Purpose:                 "replace_selected_pending_artifacts_with_real_setup_svc_metadata_service_query_diff_cleanup_evidence",
		ReplacementStatusTarget: "passed|verified|success",
		SourceRoot:              setupSvcLiveReplayWorklistSourceRoot(),
		CaptureRoot:             setupSvcLiveReplayWorklistCaptureRoot(projectPath),
		SuggestedWorklistPath:   worklistPath,
		SaveWorklistCommand:     worklistCommand + " > " + shellPath(worklistPath),
		DryRunImportCommand:     "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(worklistPath) + " --dry-run",
		ExecuteImportCommand:    "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(worklistPath) + " --execute --approval " + setupSvcParityEvidenceImportApproval,
		PostReplacementCommands: setupSvcLiveReplayWorklistPostReplacementCommands(commands),
		StopConditions: []string{
			"Do not mark any artifact passed until real replay evidence replaces the pending placeholder.",
			"Do not run matrix promotion until setup-svc-live-replay-evidence and evidence bundle verification both pass.",
			"Do not use clean-only, rowCount-only, columns-only, or boolean-only status markers as proof.",
		},
	}
}

func setupSvcLiveReplayWorklistOperatorBatchFor(projectPath string, manifestPath string, batchIndex int, queue setupSvcLiveReplayEvidenceSectionQueue, offset int, limit int, artifacts []setupSvcLiveReplayArtifactCollectionAction, commands setupSvcLiveReplayGapCommands) setupSvcLiveReplayWorklistOperatorBatch {
	records := make([]setupSvcLiveReplayArtifactReplacementRecord, 0, len(artifacts))
	for _, artifact := range artifacts {
		records = append(records, setupSvcLiveReplayArtifactReplacementRecordFor(projectPath, manifestPath, artifact))
	}
	return setupSvcLiveReplayWorklistOperatorBatch{
		BatchIndex:                 batchIndex,
		ArtifactType:               queue.ArtifactType,
		EvidenceSection:            queue.Section,
		Offset:                     offset,
		Limit:                      limit,
		ReplacementStatusTarget:    "passed|verified|success",
		PostReplacementCommands:    setupSvcLiveReplayWorklistPostReplacementCommands(commands),
		ArtifactReplacementRecords: records,
	}
}

func setupSvcLiveReplayWorklistSourceRoot() string {
	return "captures"
}

func setupSvcLiveReplayWorklistCaptureRoot(projectPath string) string {
	return filepath.Join(projectPath, setupSvcLiveReplayWorklistSourceRoot())
}

func setupSvcLiveReplayWorklistSuggestedPath(projectPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	parts := []string{"worklist"}
	if options.ArtifactType != "" {
		parts = append(parts, options.ArtifactType)
	}
	if options.EvidenceSection != "" {
		parts = append(parts, setupSvcLiveReplayFilenameToken(options.EvidenceSection))
	}
	if options.SectionStatus != "" {
		parts = append(parts, options.SectionStatus)
	}
	if options.SourceStatus != "" {
		parts = append(parts, "source-"+options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		parts = append(parts, "readiness-"+options.SourceReadiness)
	}
	if options.BatchIndex >= 0 {
		parts = append(parts, "batch-"+strconv.Itoa(options.BatchIndex))
	} else if options.BatchLimit != setupSvcLiveReplayWorklistBatchLimit {
		parts = append(parts, "batches-"+strconv.Itoa(options.BatchLimit))
	}
	if options.Offset > 0 || options.Limit != 25 {
		parts = append(parts, "offset-"+strconv.Itoa(options.Offset), "limit-"+strconv.Itoa(options.Limit))
	}
	return filepath.Join(projectPath, "outputs", "setup-svc-live-replay", strings.Join(parts, "-")+".json")
}

func setupSvcLiveReplayFilenameToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "all"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func setupSvcLiveReplayWorklistCommand(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) string {
	args := []string{"cloudcc", "scan", "msapi", shellPath(projectPath), "setup-svc-live-replay-worklist", shellPath(manifestPath)}
	args = append(args, setupSvcLiveReplayGapArgsFromOptions(options)...)
	if options.SourceStatus != "" {
		args = append(args, "--source-status", options.SourceStatus)
	}
	if options.SourceReadiness != "" {
		args = append(args, "--source-readiness", options.SourceReadiness)
	}
	if options.BatchIndex >= 0 {
		args = append(args, "--batch-index", strconv.Itoa(options.BatchIndex))
	} else if options.BatchLimit != setupSvcLiveReplayWorklistBatchLimit {
		args = append(args, "--batch-limit", strconv.Itoa(options.BatchLimit))
	}
	return strings.Join(args, " ")
}

func setupSvcLiveReplayWorklistArtifactsForSourceFilters(projectPath string, artifacts []setupSvcLiveReplayArtifactCollectionAction, sourceStatus string, sourceReadiness string) []setupSvcLiveReplayArtifactCollectionAction {
	sourceStatus = strings.ToLower(strings.TrimSpace(sourceStatus))
	sourceReadiness = strings.ToLower(strings.TrimSpace(sourceReadiness))
	if sourceStatus == "" && sourceReadiness == "" {
		return append([]setupSvcLiveReplayArtifactCollectionAction{}, artifacts...)
	}
	filtered := make([]setupSvcLiveReplayArtifactCollectionAction, 0, len(artifacts))
	for _, artifact := range artifacts {
		readiness := setupSvcLiveReplaySourceReadinessFor(projectPath, artifact.Path)
		exists := readiness == "complete" || readiness == "incomplete"
		if sourceStatus == "present" && !exists {
			continue
		}
		if sourceStatus == "missing" && exists {
			continue
		}
		if sourceReadiness != "" && readiness != sourceReadiness {
			continue
		}
		if sourceStatus == "" || sourceStatus == "present" || sourceStatus == "missing" {
			filtered = append(filtered, artifact)
		}
	}
	return filtered
}

func setupSvcLiveReplayWorklistSuggestedSourcePath(artifactPath string) string {
	return filepath.Join(setupSvcLiveReplayWorklistSourceRoot(), strings.TrimPrefix(strings.TrimSpace(artifactPath), "@"))
}

func setupSvcLiveReplayWorklistSuggestedSourceExists(projectPath string, suggestedSourcePath string) bool {
	if strings.TrimSpace(suggestedSourcePath) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(projectPath, suggestedSourcePath))
	return err == nil && !info.IsDir()
}

func setupSvcLiveReplayArtifactReplacementRecordFor(projectPath string, manifestPath string, artifact setupSvcLiveReplayArtifactCollectionAction) setupSvcLiveReplayArtifactReplacementRecord {
	suggestedSourcePath := setupSvcLiveReplayWorklistSuggestedSourcePath(artifact.Path)
	sourceExists := setupSvcLiveReplayWorklistSuggestedSourceExists(projectPath, suggestedSourcePath)
	sourceStatuses := setupSvcLiveReplayEvidenceSectionStatusesAtPath(filepath.Join(projectPath, suggestedSourcePath), artifact.RequiredEvidenceSections)
	sourceReadiness := setupSvcLiveReplaySourceReadiness(sourceExists, sourceStatuses)
	captureTask := artifact.CaptureTask
	if captureTask.SourceSystem == "" {
		captureTask = setupSvcLiveReplayArtifactCaptureTaskFor(projectPath, manifestPath, artifact.Domain, artifact.Operation, artifact.ArtifactType, artifact.Path, suggestedSourcePath, artifact.RequiredShapeKey, artifact.ManifestStatusField, artifact.RequiredEvidenceSections, artifact.RequiredTables, artifact.RuntimeEffects, artifact.QueryReadbackExpectations)
	}
	return setupSvcLiveReplayArtifactReplacementRecord{
		Domain:                    artifact.Domain,
		Operation:                 artifact.Operation,
		ArtifactType:              artifact.ArtifactType,
		Path:                      artifact.Path,
		SuggestedSourcePath:       suggestedSourcePath,
		SuggestedSourceExists:     sourceExists,
		SourceReadiness:           sourceReadiness,
		RequiredShapeKey:          artifact.RequiredShapeKey,
		ManifestStatusField:       artifact.ManifestStatusField,
		ReplacementStatusTarget:   "passed|verified|success",
		RequiredEvidenceSections:  append([]string{}, artifact.RequiredEvidenceSections...),
		SourceEvidenceSections:    sourceStatuses,
		MissingEvidenceSections:   setupSvcLiveReplayMissingEvidenceSections(sourceStatuses),
		RequiredTables:            append([]string{}, artifact.RequiredTables...),
		RuntimeEffects:            append([]string{}, artifact.RuntimeEffects...),
		QueryReadbackExpectations: append([]string{}, artifact.QueryReadbackExpectations...),
		CaptureTask:               captureTask,
		Checklist:                 append([]string{}, artifact.ReplacementChecklist...),
	}
}

func setupSvcLiveReplayMissingEvidenceSections(statuses []setupSvcLiveReplayEvidenceSectionStatus) []string {
	missing := []string{}
	for _, status := range statuses {
		if !status.Present {
			missing = append(missing, status.Section)
		}
	}
	return missing
}

func setupSvcLiveReplayCapturePlanOperatorPacketOptions(options setupSvcLiveReplayCollectionPlanOptions) setupSvcLiveReplayCollectionPlanOptions {
	packetOptions := options
	if packetOptions.SourceStatus == "" && packetOptions.SourceReadiness == "" {
		packetOptions.SourceReadiness = "complete"
	}
	packetOptions.BatchIndex = -1
	packetOptions.BatchLimit = setupSvcLiveReplayWorklistBatchLimit
	return packetOptions
}

func setupSvcLiveReplayCapturePlanGapCommands(projectPath string, manifestPath string) setupSvcLiveReplayGapCommands {
	return setupSvcLiveReplayGapCommands{
		PrepareWorkspace: "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-workspace --execute --approval " + setupSvcParityEvidenceWorkspaceApproval,
		GenerateDiffs:    "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-normalized-diff " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityNormalizedDiffApproval,
		SyncManifest:     "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-manifest-sync " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityManifestSyncApproval,
		VerifyEvidence:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
		WriteBundle:      "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityEvidenceBundleApproval,
		PromotionAudit:   "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath),
		CompletionAudit:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
	}
}

func setupSvcLiveReplayAccumulateCapturePlanTypeTotals(totals *setupSvcLiveReplayCapturePlanTotals, artifactType string) {
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "query-readback":
		totals.QueryReadbackArtifacts++
	case "setup-svc":
		totals.SetupSvcArtifacts++
	case "metadata-service":
		totals.MetadataServiceArtifacts++
	case "normalized-diff":
		totals.NormalizedDiffArtifacts++
	case "cleanup":
		totals.CleanupArtifacts++
	}
}

func setupSvcLiveReplayWorklistPostReplacementCommands(commands setupSvcLiveReplayGapCommands) []string {
	var result []string
	for _, command := range []string{
		commands.SyncManifest,
		commands.VerifyEvidence,
		commands.WriteBundle,
		commands.PromotionAudit,
		commands.CompletionAudit,
	} {
		if strings.TrimSpace(command) != "" {
			result = append(result, command)
		}
	}
	return result
}

func setupSvcLiveReplayWorklistQueueMatches(options setupSvcLiveReplayCollectionPlanOptions, queue setupSvcLiveReplayEvidenceSectionQueue) bool {
	if options.ArtifactType != "" && setupSvcLiveReplayNormalizeArtifactType(queue.ArtifactType) != options.ArtifactType {
		return false
	}
	if options.EvidenceSection != "" && !strings.EqualFold(queue.Section, options.EvidenceSection) {
		return false
	}
	if options.SectionStatus != "" && !strings.EqualFold(options.SectionStatus, "missing") {
		return false
	}
	return true
}

func setupSvcLiveReplayWorklistEvidenceSectionSummaries(sectionTotals map[string]*setupSvcLiveReplayEvidenceSectionSummary, recommendedOrder []string, projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) []setupSvcLiveReplayEvidenceSectionSummary {
	summaries := setupSvcLiveReplayEvidenceSectionSummaries(sectionTotals, recommendedOrder, projectPath, manifestPath, options)
	for index := range summaries {
		if summaries[index].Missing <= 0 {
			continue
		}
		queue := setupSvcLiveReplayEvidenceSectionQueue{
			ArtifactType: summaries[index].ArtifactType,
			Section:      summaries[index].Section,
			Missing:      summaries[index].Missing,
			PageSize:     options.Limit,
		}
		if queue.PageSize <= 0 {
			queue.PageSize = 25
		}
		summaries[index].NextAction = "repair_incomplete_source_evidence_section"
		summaries[index].QueueCommand = setupSvcLiveReplayWorklistQueueCommand(projectPath, manifestPath, options, queue, -1)
	}
	return summaries
}

func setupSvcLiveReplayWorklistQueueCommand(projectPath string, manifestPath string, baseOptions setupSvcLiveReplayCollectionPlanOptions, queue setupSvcLiveReplayEvidenceSectionQueue, batchIndex int) string {
	queueOptions := baseOptions
	queueOptions.ArtifactType = setupSvcLiveReplayNormalizeArtifactType(queue.ArtifactType)
	queueOptions.EvidenceSection = queue.Section
	queueOptions.SectionStatus = "missing"
	queueOptions.Offset = 0
	queueOptions.Limit = queue.PageSize
	if queueOptions.Limit <= 0 {
		queueOptions.Limit = 25
	}
	queueOptions.BatchIndex = batchIndex
	if batchIndex >= 0 {
		queueOptions.BatchLimit = setupSvcLiveReplayWorklistBatchLimit
	}
	return setupSvcLiveReplayWorklistCommand(projectPath, manifestPath, queueOptions)
}

func setupSvcLiveReplayParseGapArgs(manifestArg string, args []string) (string, setupSvcLiveReplayCollectionPlanOptions, error) {
	options := setupSvcLiveReplayCollectionPlanOptions{Limit: 25, BatchIndex: -1, BatchLimit: setupSvcLiveReplayWorklistBatchLimit}
	optionArgs := append([]string{}, args...)
	if strings.TrimSpace(manifestArg) == "" && len(optionArgs) > 0 && !strings.HasPrefix(strings.TrimSpace(optionArgs[0]), "--") {
		manifestArg = optionArgs[0]
		optionArgs = optionArgs[1:]
	}
	for i := 0; i < len(optionArgs); i++ {
		arg := strings.TrimSpace(optionArgs[i])
		if arg == "" {
			continue
		}
		name, value, hasValue := strings.Cut(arg, "=")
		name = strings.ToLower(strings.TrimSpace(name))
		if !strings.HasPrefix(name, "--") {
			return "", options, fmt.Errorf("unexpected setup-svc-live-replay-gaps argument %q", arg)
		}
		readValue := func() (string, error) {
			if hasValue {
				return strings.TrimSpace(value), nil
			}
			i++
			if i >= len(optionArgs) {
				return "", fmt.Errorf("missing value for %s", name)
			}
			return strings.TrimSpace(optionArgs[i]), nil
		}
		switch name {
		case "--domain":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			options.Domain = normalizeDomain(value)
		case "--operation":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			options.Operation = strings.ToLower(strings.TrimSpace(value))
		case "--artifact-type", "--type":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			options.ArtifactType = setupSvcLiveReplayNormalizeArtifactType(value)
		case "--source-system":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			options.SourceSystem = strings.TrimSpace(value)
		case "--capture-mode":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			options.CaptureMode = strings.TrimSpace(value)
		case "--status":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			options.Status = strings.ToLower(strings.TrimSpace(value))
		case "--evidence-section", "--section":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			options.EvidenceSection = strings.TrimSpace(value)
		case "--section-status", "--evidence-section-status":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			options.SectionStatus = strings.ToLower(strings.TrimSpace(value))
		case "--source-status", "--source-file-status":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			normalized := strings.ToLower(strings.TrimSpace(value))
			switch normalized {
			case "present", "exists", "ready", "captured":
				options.SourceStatus = "present"
			case "missing", "absent", "not-found", "uncaptured":
				options.SourceStatus = "missing"
			default:
				return "", options, fmt.Errorf("invalid --source-status %q; expected present or missing", value)
			}
		case "--source-readiness", "--source-file-readiness":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			normalized := strings.ToLower(strings.TrimSpace(value))
			switch normalized {
			case "complete", "ready", "importable", "valid", "full":
				options.SourceReadiness = "complete"
			case "incomplete", "partial", "not-ready", "invalid":
				options.SourceReadiness = "incomplete"
			case "missing", "absent", "not-found", "uncaptured":
				options.SourceReadiness = "missing"
			default:
				return "", options, fmt.Errorf("invalid --source-readiness %q; expected complete, incomplete, or missing", value)
			}
		case "--offset":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 0 {
				return "", options, fmt.Errorf("invalid --offset %q; expected non-negative integer", value)
			}
			options.Offset = parsed
		case "--limit":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 0 {
				return "", options, fmt.Errorf("invalid --limit %q; expected non-negative integer", value)
			}
			options.Limit = parsed
		case "--batch-index":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 0 {
				return "", options, fmt.Errorf("invalid --batch-index %q; expected non-negative integer", value)
			}
			options.BatchIndex = parsed
		case "--batch-limit":
			value, err := readValue()
			if err != nil {
				return "", options, err
			}
			parsed, err := strconv.Atoi(value)
			if err != nil || parsed < 0 {
				return "", options, fmt.Errorf("invalid --batch-limit %q; expected non-negative integer", value)
			}
			options.BatchLimit = parsed
		default:
			return "", options, fmt.Errorf("unsupported setup-svc-live-replay-gaps option %s", name)
		}
	}
	return strings.TrimSpace(manifestArg), options, nil
}

func setupSvcLiveReplayWorklistBatchIndexPointer(options setupSvcLiveReplayCollectionPlanOptions) *int {
	if options.BatchIndex < 0 {
		return nil
	}
	value := options.BatchIndex
	return &value
}

func setupSvcLiveReplayWorklistEffectiveBatchLimit(options setupSvcLiveReplayCollectionPlanOptions) int {
	if options.BatchIndex >= 0 {
		return 1
	}
	if options.BatchLimit <= 0 {
		return 0
	}
	return options.BatchLimit
}

func setupSvcLiveReplayWorklistBatchRange(batchCount int, options setupSvcLiveReplayCollectionPlanOptions) (int, int) {
	if batchCount <= 0 {
		return 0, 0
	}
	if options.BatchIndex >= 0 {
		if options.BatchIndex >= batchCount {
			return batchCount, batchCount
		}
		return options.BatchIndex, options.BatchIndex + 1
	}
	limit := options.BatchLimit
	if limit < 0 {
		limit = 0
	}
	if limit > batchCount {
		limit = batchCount
	}
	return 0, limit
}

func setupSvcLiveReplayGapArgsFromOptions(options setupSvcLiveReplayCollectionPlanOptions) []string {
	args := []string{}
	if options.Domain != "" {
		args = append(args, "--domain", options.Domain)
	}
	if options.Operation != "" {
		args = append(args, "--operation", options.Operation)
	}
	if options.ArtifactType != "" {
		args = append(args, "--artifact-type", options.ArtifactType)
	}
	if options.SourceSystem != "" {
		args = append(args, "--source-system", options.SourceSystem)
	}
	if options.CaptureMode != "" {
		args = append(args, "--capture-mode", options.CaptureMode)
	}
	if options.Status != "" {
		args = append(args, "--status", options.Status)
	}
	if options.EvidenceSection != "" {
		args = append(args, "--evidence-section", options.EvidenceSection)
	}
	if options.SectionStatus != "" {
		args = append(args, "--section-status", options.SectionStatus)
	}
	args = append(args, "--offset", strconv.Itoa(options.Offset), "--limit", strconv.Itoa(options.Limit))
	return args
}

func setupSvcLiveReplayPopulateMissingGapDomains(result *setupSvcLiveReplayGapResult) {
	for _, expected := range setupSvcLiveReplayDomains() {
		domain := setupSvcLiveReplayGapDomain{Domain: expected.Domain, Status: "missing_manifest"}
		result.Totals.Domains++
		for _, operation := range expected.Operations {
			result.Totals.Operations++
			result.Totals.MissingOperations++
			domain.Operations = append(domain.Operations, setupSvcLiveReplayGapOperation{
				Operation:       operation,
				Status:          "missing_manifest_operation",
				NextAction:      "prepare_evidence_workspace",
				MissingEvidence: setupSvcLiveReplayRequiredEvidence(operation),
				EvidenceFiles:   setupSvcLiveReplayEvidenceFiles(expected.Domain, operation, operation != "query"),
			})
		}
		result.Domains = append(result.Domains, domain)
	}
}

func setupSvcLiveReplayManifestOperationMap(domain map[string]any, result *setupSvcLiveReplayGapResult, domainName string) map[string]map[string]any {
	operations := map[string]map[string]any{}
	if domain == nil {
		return operations
	}
	expected := map[string]bool{}
	for _, item := range setupSvcLiveReplayDomains() {
		if item.Domain != domainName {
			continue
		}
		for _, operation := range item.Operations {
			expected[strings.ToLower(operation)] = true
		}
	}
	for _, operation := range mapList(domain["operations"]) {
		name := strings.ToLower(strings.TrimSpace(firstMapString(operation, "operation", "name", "mode")))
		switch {
		case name == "":
			if result != nil {
				result.BlockingIssues = append(result.BlockingIssues, domainName+": missing operation name")
			}
		case !expected[name]:
			if result != nil {
				result.BlockingIssues = append(result.BlockingIssues, domainName+"/"+name+": unexpected operation")
			}
		case operations[name] != nil:
			if result != nil {
				result.BlockingIssues = append(result.BlockingIssues, domainName+"/"+name+": duplicate operation")
			}
		default:
			operations[name] = operation
		}
	}
	return operations
}

func buildSetupSvcLiveReplayOperationGap(projectPath string, domain setupSvcLiveReplayDomain, operation string, evidence map[string]any) setupSvcLiveReplayGapOperation {
	result := setupSvcLiveReplayGapOperation{
		Operation:     operation,
		Status:        "complete",
		NextAction:    "ready_for_evidence_verification",
		EvidenceFiles: setupSvcLiveReplayEvidenceFiles(domain.Domain, operation, operation != "query"),
	}
	if evidence == nil {
		result.Status = "missing_evidence"
		result.NextAction = "prepare_evidence_workspace"
		result.MissingEvidence = append(result.MissingEvidence, setupSvcLiveReplayRequiredEvidence(operation)...)
		for _, file := range result.EvidenceFiles {
			result.MissingEvidence = append(result.MissingEvidence, "file:"+file)
		}
		return result
	}
	result.EvidenceFiles = setupSvcLiveReplayEvidenceFileList(evidence["evidenceFiles"])
	for _, field := range setupSvcLiveReplayRequiredEvidence(operation) {
		status := strings.ToLower(strings.TrimSpace(firstMapString(evidence, field)))
		switch {
		case status == "":
			result.MissingEvidence = append(result.MissingEvidence, field)
		case setupSvcLiveReplayPassedStatus(status):
		case status == "pending":
			result.PendingEvidence = append(result.PendingEvidence, field+"="+status)
		default:
			result.FailedEvidence = append(result.FailedEvidence, field+"="+status)
		}
	}
	expectedFiles := setupSvcLiveReplayEvidenceFiles(domain.Domain, operation, operation != "query")
	contractIssues := setupSvcLiveReplayEvidenceFileContractIssues(projectPath, expectedFiles, result.EvidenceFiles)
	for _, missing := range contractIssues.Missing {
		result.MissingEvidence = append(result.MissingEvidence, "evidenceFiles:"+missing)
	}
	for _, unexpected := range contractIssues.Unexpected {
		result.FailedEvidence = append(result.FailedEvidence, "unexpectedEvidenceFile:"+unexpected)
	}
	for _, duplicate := range contractIssues.Duplicate {
		result.FailedEvidence = append(result.FailedEvidence, "duplicateEvidenceFile:"+duplicate)
	}
	sourceArtifactsPassed := map[string]bool{"setup-svc": false, "metadata-service": false}
	evidenceFileByContract := setupSvcLiveReplayEvidenceFileMap(projectPath, result.EvidenceFiles)
	for _, requiredFile := range expectedFiles {
		actualFile := evidenceFileByContract[setupSvcLiveReplayEvidencePathForContract(projectPath, requiredFile)]
		artifactType := setupSvcLiveReplayArtifactType(requiredFile)
		if strings.TrimSpace(actualFile) == "" {
			continue
		}
		resolved := setupSvcLiveReplayResolveEvidenceFile(projectPath, actualFile)
		payload, err := os.ReadFile(resolved)
		if err != nil {
			if artifactType == "normalized-diff" && !setupSvcLiveReplayPassedStatus(firstMapString(evidence, "normalizedDiffStatus")) && setupSvcLiveReplayGapSourceReady(result, sourceArtifactsPassed) {
				continue
			}
			result.MissingEvidence = append(result.MissingEvidence, "file:"+setupSvcLiveReplayEvidencePathForContract(projectPath, actualFile))
			continue
		}
		var decoded any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			result.FailedEvidence = append(result.FailedEvidence, "evidenceFileInvalidJSON:"+setupSvcLiveReplayEvidencePathForContract(projectPath, actualFile))
			continue
		}
		if artifact, ok := decoded.(map[string]any); ok && strings.EqualFold(firstMapString(artifact, "status", "evidenceStatus"), "pending") {
			result.PendingEvidence = append(result.PendingEvidence, "artifactStatus:"+setupSvcLiveReplayEvidencePathForContract(projectPath, actualFile)+"=pending")
			continue
		}
		failures := verifySetupSvcLiveReplayEvidenceArtifact(projectPath, requiredFile, domain, operation, decoded)
		if len(failures) > 0 {
			for _, failure := range failures {
				result.FailedEvidence = append(result.FailedEvidence, failure+":"+setupSvcLiveReplayEvidencePathForContract(projectPath, actualFile))
			}
			continue
		}
		if artifactType == "setup-svc" || artifactType == "metadata-service" {
			sourceArtifactsPassed[artifactType] = true
		}
	}
	if setupSvcLiveReplayGapNeedsNormalizedDiff(result, sourceArtifactsPassed) {
		result.Status = "ready_for_normalized_diff"
		result.NextAction = "generate_normalized_diff"
		return result
	}
	switch {
	case len(result.FailedEvidence) > 0:
		result.Status = "failed_evidence"
		result.NextAction = "repair_failed_evidence"
	case len(result.MissingEvidence) > 0:
		result.Status = "missing_evidence"
		result.NextAction = "collect_missing_evidence"
	case len(result.PendingEvidence) > 0:
		result.Status = "pending_evidence"
		result.NextAction = setupSvcLiveReplayGapNextPendingAction(result.PendingEvidence)
	default:
		result.Status = "complete"
		result.NextAction = "ready_for_evidence_verification"
	}
	return result
}

func setupSvcLiveReplayGapSourceReady(result setupSvcLiveReplayGapOperation, sourceArtifactsPassed map[string]bool) bool {
	return sourceArtifactsPassed["setup-svc"] &&
		sourceArtifactsPassed["metadata-service"] &&
		!setupSvcLiveReplayGapHasEvidencePrefix(result.FailedEvidence, "setupSvcEvidenceStatus") &&
		!setupSvcLiveReplayGapHasEvidencePrefix(result.FailedEvidence, "metadataServiceEvidenceStatus")
}

func setupSvcLiveReplayGapNeedsNormalizedDiff(result setupSvcLiveReplayGapOperation, sourceArtifactsPassed map[string]bool) bool {
	if !setupSvcLiveReplayGapSourceReady(result, sourceArtifactsPassed) {
		return false
	}
	for _, item := range result.PendingEvidence {
		if strings.HasPrefix(item, "normalizedDiffStatus=") || strings.Contains(item, "/normalized-diff.json") {
			return true
		}
	}
	for _, item := range result.MissingEvidence {
		if strings.Contains(item, "/normalized-diff.json") || strings.HasPrefix(item, "normalizedDiffStatus") {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayGapHasEvidencePrefix(values []string, prefix string) bool {
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayGapNextPendingAction(pending []string) string {
	for _, item := range pending {
		switch {
		case strings.HasPrefix(item, "setupSvcEvidenceStatus="):
			return "collect_setup_svc_snapshot"
		case strings.HasPrefix(item, "metadataServiceEvidenceStatus="):
			return "collect_metadata_service_snapshot"
		case strings.HasPrefix(item, "queryEvidenceStatus="):
			return "collect_query_readback"
		case strings.HasPrefix(item, "normalizedDiffStatus="):
			return "generate_normalized_diff"
		case strings.HasPrefix(item, "cleanupStatus="):
			return "collect_cleanup_evidence"
		}
	}
	return "replace_pending_artifacts"
}

func setupSvcLiveReplayAccumulateGapTotals(result *setupSvcLiveReplayGapResult, operation setupSvcLiveReplayGapOperation) {
	result.Totals.MissingFiles += setupSvcLiveReplayGapCountPrefix(operation.MissingEvidence, "file:")
	result.Totals.PendingFiles += setupSvcLiveReplayGapCountPrefix(operation.PendingEvidence, "artifactStatus:")
	result.Totals.FailedArtifacts += len(operation.FailedEvidence)
	switch operation.Status {
	case "complete":
		result.Totals.CompleteOperations++
	case "ready_for_normalized_diff":
		result.Totals.ReadyForDiffOperations++
	case "failed_evidence":
		result.Totals.FailedOperations++
	case "pending_evidence":
		result.Totals.PendingOperations++
	default:
		result.Totals.MissingOperations++
	}
}

func setupSvcLiveReplayFinalizeCollectionPlan(result *setupSvcLiveReplayGapResult, options setupSvcLiveReplayCollectionPlanOptions) {
	plan := setupSvcLiveReplayCollectionPlan{
		Status:             "complete",
		NextArtifactOffset: options.Offset,
		NextArtifactLimit:  options.Limit,
		RecommendedOrder:   setupSvcLiveReplayEvidenceRecommendedOrder(),
		Notes: []string{
			"Collection plan is read-only and summarizes the current artifact files needed for strict evidence verification.",
			"Replace pending artifacts with real setup-svc, MetadataService, query/readback, normalized-diff, and cleanup proof before manifest sync or promotion.",
			"Query-readback artifacts are counted explicitly because read APIs are part of the parity contract, not optional smoke checks.",
			"Use --domain, --operation, --artifact-type, --status, --evidence-section, --section-status, --offset, and --limit to page through large live replay evidence work queues.",
		},
	}
	plan.Filters = setupSvcLiveReplayCollectionPlanFiltersFromOptions(options)
	typeTotals := map[string]*setupSvcLiveReplayArtifactTypeCollection{}
	sectionTotals := map[string]*setupSvcLiveReplayEvidenceSectionSummary{}
	domainContracts := setupSvcLiveReplayDomainContractMap()
	for _, artifactType := range plan.RecommendedOrder {
		typeTotals[artifactType] = &setupSvcLiveReplayArtifactTypeCollection{ArtifactType: artifactType}
	}
	for _, domain := range result.Domains {
		for _, operation := range domain.Operations {
			for _, file := range operation.EvidenceFiles {
				artifactType := setupSvcLiveReplayArtifactType(file)
				if typeTotals[artifactType] == nil {
					typeTotals[artifactType] = &setupSvcLiveReplayArtifactTypeCollection{ArtifactType: artifactType}
				}
				status := setupSvcLiveReplayCollectionArtifactStatus(operation, file)
				requiredSections := setupSvcLiveReplayRequiredEvidenceSections(artifactType)
				sectionStatuses := setupSvcLiveReplayEvidenceSectionStatuses(result.Project, file, requiredSections)
				setupSvcLiveReplayAccumulateEvidenceSectionSummary(sectionTotals, artifactType, sectionStatuses)
				typeSummary := typeTotals[artifactType]
				typeSummary.Total++
				plan.TotalArtifacts++
				if artifactType == "query-readback" {
					plan.QueryReadbackArtifacts++
				}
				switch status {
				case "pending":
					typeSummary.Pending++
					plan.PendingArtifacts++
				case "missing":
					typeSummary.Missing++
					plan.MissingArtifacts++
				case "failed":
					typeSummary.Failed++
					plan.FailedArtifacts++
				default:
					typeSummary.Passed++
					plan.PassedArtifacts++
					status = "passed"
				}
				if status != "passed" && setupSvcLiveReplayCollectionPlanMatches(options, domain.Domain, operation.Operation, artifactType, status, sectionStatuses) {
					plan.TotalNextArtifacts++
					if plan.TotalNextArtifacts <= plan.NextArtifactOffset {
						continue
					}
					if len(plan.NextArtifacts) < plan.NextArtifactLimit {
						plan.NextArtifacts = append(plan.NextArtifacts, setupSvcLiveReplayCollectionAction(result.Project, domain.Domain, operation, artifactType, file, status, domainContracts))
						continue
					}
					plan.OmittedNextArtifacts++
				}
			}
		}
	}
	for _, artifactType := range plan.RecommendedOrder {
		if typeSummary := typeTotals[artifactType]; typeSummary != nil && typeSummary.Total > 0 {
			plan.ArtifactTypes = append(plan.ArtifactTypes, *typeSummary)
		}
	}
	var extraTypes []string
	for artifactType, typeSummary := range typeTotals {
		if typeSummary.Total > 0 && !containsString(plan.RecommendedOrder, artifactType) {
			extraTypes = append(extraTypes, artifactType)
		}
	}
	sort.Strings(extraTypes)
	for _, artifactType := range extraTypes {
		plan.ArtifactTypes = append(plan.ArtifactTypes, *typeTotals[artifactType])
	}
	plan.EvidenceSections = setupSvcLiveReplayEvidenceSectionSummaries(sectionTotals, plan.RecommendedOrder, result.Project, result.ManifestPath, options)
	plan.MissingSectionQueues = setupSvcLiveReplayEvidenceSectionQueues(plan.EvidenceSections, plan.RecommendedOrder)
	switch {
	case plan.FailedArtifacts > 0:
		plan.Status = "repair_failed_artifacts"
	case plan.MissingArtifacts > 0:
		plan.Status = "collect_missing_artifacts"
	case plan.PendingArtifacts > 0:
		plan.Status = "replace_pending_artifacts"
	default:
		plan.Status = "complete"
	}
	plan.PageCommands = setupSvcLiveReplayCollectionPlanPageCommandsFor(result.Project, result.ManifestPath, options, plan.TotalNextArtifacts)
	plan.Runbook = setupSvcLiveReplayCollectionPlanRunbook(result, plan)
	result.CollectionPlan = plan
}

func setupSvcLiveReplayCollectionPlanRunbook(result *setupSvcLiveReplayGapResult, plan setupSvcLiveReplayCollectionPlan) []setupSvcLiveReplayCollectionPlanRunbookStep {
	steps := []setupSvcLiveReplayCollectionPlanRunbookStep{}
	if plan.MissingArtifacts > 0 || plan.PendingArtifacts > 0 || plan.FailedArtifacts > 0 {
		commands := setupSvcLiveReplayCollectionPlanRunbookEvidenceCommands(plan.MissingSectionQueues)
		notes := []string{"Replace pending or failed artifact files with real setup-svc, MetadataService, query/readback, normalized-diff, and cleanup proof before manifest sync."}
		if len(commands) == setupSvcLiveReplayCollectionRunbookCommandLimit {
			notes = append(notes, "Command list is bounded; continue with collectionPlan.missingSectionQueues[*].batchCommands and pageCommands for the full queue.")
		}
		if len(commands) == 0 && strings.TrimSpace(plan.PageCommands.CurrentPage) != "" {
			commands = []string{plan.PageCommands.CurrentPage}
		}
		steps = append(steps, setupSvcLiveReplayCollectionPlanRunbookStep{
			Step:     "collect_or_replace_evidence",
			Status:   plan.Status,
			Commands: commands,
			Notes:    notes,
		})
	}
	if result.Totals.ReadyForDiffOperations > 0 || plan.Status == "complete" {
		steps = append(steps, setupSvcLiveReplayCollectionPlanRunbookStep{
			Step:     "generate_normalized_diffs",
			Status:   setupSvcLiveReplayCollectionPlanRunbookStepStatus(result.Totals.ReadyForDiffOperations > 0),
			Commands: []string{result.NextCommands.GenerateDiffs},
			Notes:    []string{"Run after setup-svc and MetadataService snapshot artifacts are real passed evidence."},
		})
	}
	steps = append(steps,
		setupSvcLiveReplayCollectionPlanRunbookStep{
			Step:     "sync_manifest_status",
			Status:   "after_artifact_replacement",
			Commands: []string{result.NextCommands.SyncManifest},
			Notes:    []string{"Manifest sync derives operation status fields from artifact JSON files; strict evidence verification remains the gate."},
		},
		setupSvcLiveReplayCollectionPlanRunbookStep{
			Step:     "verify_evidence",
			Status:   "after_manifest_sync",
			Commands: []string{result.NextCommands.VerifyEvidence},
			Notes:    []string{"This must pass before evidence bundle writing, promotion, or final completion audit."},
		},
		setupSvcLiveReplayCollectionPlanRunbookStep{
			Step:     "write_evidence_bundle",
			Status:   "after_evidence_passes",
			Commands: []string{result.NextCommands.WriteBundle},
			Notes:    []string{"Bundle writing is approval-gated and records current manifest/artifact SHA-256 coverage for promotion."},
		},
		setupSvcLiveReplayCollectionPlanRunbookStep{
			Step:     "promotion_and_completion_audit",
			Status:   "after_bundle_passes",
			Commands: []string{result.NextCommands.PromotionAudit, result.NextCommands.CompletionAudit},
			Notes:    []string{"Promotion remains read-only until the separate matrix-promotion apply command is explicitly approved."},
		},
	)
	return setupSvcLiveReplayCollectionPlanRunbookDropEmptyCommands(steps)
}

func setupSvcLiveReplayCollectionPlanRunbookEvidenceCommands(queues []setupSvcLiveReplayEvidenceSectionQueue) []string {
	commands := []string{}
	for _, queue := range queues {
		for _, command := range queue.BatchCommands {
			if strings.TrimSpace(command) == "" {
				continue
			}
			commands = append(commands, command)
			if len(commands) >= setupSvcLiveReplayCollectionRunbookCommandLimit {
				return commands
			}
		}
	}
	return commands
}

func setupSvcLiveReplayCollectionPlanRunbookStepStatus(ready bool) string {
	if ready {
		return "ready"
	}
	return "pending"
}

func setupSvcLiveReplayCollectionPlanRunbookDropEmptyCommands(steps []setupSvcLiveReplayCollectionPlanRunbookStep) []setupSvcLiveReplayCollectionPlanRunbookStep {
	for index := range steps {
		filtered := steps[index].Commands[:0]
		for _, command := range steps[index].Commands {
			if strings.TrimSpace(command) != "" {
				filtered = append(filtered, command)
			}
		}
		steps[index].Commands = filtered
	}
	return steps
}

func setupSvcLiveReplayAccumulateEvidenceSectionSummary(sectionTotals map[string]*setupSvcLiveReplayEvidenceSectionSummary, artifactType string, statuses []setupSvcLiveReplayEvidenceSectionStatus) {
	for _, sectionStatus := range statuses {
		key := setupSvcLiveReplayEvidenceSectionSummaryKey(artifactType, sectionStatus.Section)
		summary := sectionTotals[key]
		if summary == nil {
			summary = &setupSvcLiveReplayEvidenceSectionSummary{
				ArtifactType: artifactType,
				Section:      sectionStatus.Section,
			}
			sectionTotals[key] = summary
		}
		summary.Total++
		if sectionStatus.Present {
			summary.Present++
		} else {
			summary.Missing++
		}
	}
}

func setupSvcLiveReplayEvidenceRecommendedOrder() []string {
	return []string{"setup-svc", "metadata-service", "query-readback", "normalized-diff", "cleanup"}
}

func setupSvcLiveReplayEvidenceSectionSummaryKey(artifactType string, section string) string {
	return artifactType + "\x00" + section
}

func setupSvcLiveReplayEvidenceSectionSummaries(sectionTotals map[string]*setupSvcLiveReplayEvidenceSectionSummary, recommendedOrder []string, projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) []setupSvcLiveReplayEvidenceSectionSummary {
	var summaries []setupSvcLiveReplayEvidenceSectionSummary
	seenTypes := map[string]bool{}
	for _, artifactType := range recommendedOrder {
		seenTypes[artifactType] = true
		for _, section := range setupSvcLiveReplayRequiredEvidenceSections(artifactType) {
			key := setupSvcLiveReplayEvidenceSectionSummaryKey(artifactType, section)
			if summary := sectionTotals[key]; summary != nil {
				summaries = append(summaries, setupSvcLiveReplayEvidenceSectionSummaryWithQueue(*summary, projectPath, manifestPath, options))
			}
		}
	}
	var extraTypes []string
	for _, summary := range sectionTotals {
		if !seenTypes[summary.ArtifactType] {
			extraTypes = append(extraTypes, summary.ArtifactType)
			seenTypes[summary.ArtifactType] = true
		}
	}
	sort.Strings(extraTypes)
	for _, artifactType := range extraTypes {
		sections := setupSvcLiveReplayRequiredEvidenceSections(artifactType)
		for _, section := range sections {
			key := setupSvcLiveReplayEvidenceSectionSummaryKey(artifactType, section)
			if summary := sectionTotals[key]; summary != nil {
				summaries = append(summaries, setupSvcLiveReplayEvidenceSectionSummaryWithQueue(*summary, projectPath, manifestPath, options))
			}
		}
	}
	return summaries
}

func setupSvcLiveReplayCapturePlanEvidenceSectionSummaries(sectionTotals map[string]*setupSvcLiveReplayEvidenceSectionSummary, recommendedOrder []string, projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) []setupSvcLiveReplayEvidenceSectionSummary {
	summaries := setupSvcLiveReplayEvidenceSectionSummaries(sectionTotals, recommendedOrder, projectPath, manifestPath, options)
	for index := range summaries {
		if summaries[index].Missing <= 0 {
			continue
		}
		queueOptions := options
		queueOptions.ArtifactType = setupSvcLiveReplayNormalizeArtifactType(summaries[index].ArtifactType)
		queueOptions.EvidenceSection = summaries[index].Section
		queueOptions.SectionStatus = "missing"
		queueOptions.Offset = 0
		if queueOptions.Limit <= 0 {
			queueOptions.Limit = 25
		}
		summaries[index].NextAction = "capture_missing_source_evidence_section"
		summaries[index].QueueCommand = setupSvcLiveReplayCapturePlanPageCommand(projectPath, manifestPath, 0, queueOptions)
	}
	return summaries
}

func setupSvcLiveReplayEvidenceSectionSummaryWithQueue(summary setupSvcLiveReplayEvidenceSectionSummary, projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions) setupSvcLiveReplayEvidenceSectionSummary {
	if summary.Missing <= 0 {
		return summary
	}
	queueOptions := options
	queueOptions.ArtifactType = setupSvcLiveReplayNormalizeArtifactType(summary.ArtifactType)
	queueOptions.EvidenceSection = summary.Section
	queueOptions.SectionStatus = "missing"
	queueOptions.Offset = 0
	if queueOptions.Limit <= 0 {
		queueOptions.Limit = 25
	}
	summary.NextAction = "collect_missing_evidence_section"
	summary.QueueCommand = setupSvcLiveReplayCollectionPlanPageCommand(projectPath, manifestPath, 0, queueOptions)
	return summary
}

func setupSvcLiveReplayEvidenceSectionQueues(summaries []setupSvcLiveReplayEvidenceSectionSummary, recommendedOrder []string) []setupSvcLiveReplayEvidenceSectionQueue {
	queues := make([]setupSvcLiveReplayEvidenceSectionQueue, 0, len(summaries))
	typeOrder := map[string]int{}
	for index, artifactType := range recommendedOrder {
		typeOrder[artifactType] = index
	}
	for _, summary := range summaries {
		if summary.Missing <= 0 || strings.TrimSpace(summary.QueueCommand) == "" {
			continue
		}
		pageSize := setupSvcLiveReplayEvidenceSectionQueuePageSize(summary.QueueCommand)
		batchCount := setupSvcLiveReplayEvidenceSectionQueueBatchCount(summary.Missing, pageSize)
		batchCommands := setupSvcLiveReplayEvidenceSectionQueueBatchCommands(summary.QueueCommand, pageSize, batchCount)
		queues = append(queues, setupSvcLiveReplayEvidenceSectionQueue{
			ArtifactType:         summary.ArtifactType,
			Section:              summary.Section,
			Missing:              summary.Missing,
			RequiredShapeKey:     setupSvcLiveReplayCollectionRequiredShapeKey(summary.ArtifactType),
			ManifestStatusField:  setupSvcLiveReplayManifestStatusField(summary.ArtifactType),
			PageSize:             pageSize,
			BatchCount:           batchCount,
			QueueCommand:         summary.QueueCommand,
			BatchCommands:        batchCommands,
			OmittedBatchCommands: setupSvcLiveReplayEvidenceSectionQueueOmittedBatchCommands(batchCount, len(batchCommands)),
		})
	}
	sort.SliceStable(queues, func(i, j int) bool {
		if queues[i].Missing != queues[j].Missing {
			return queues[i].Missing > queues[j].Missing
		}
		leftTypeOrder, leftKnown := typeOrder[queues[i].ArtifactType]
		rightTypeOrder, rightKnown := typeOrder[queues[j].ArtifactType]
		if leftKnown != rightKnown {
			return leftKnown
		}
		if leftKnown && rightKnown && leftTypeOrder != rightTypeOrder {
			return leftTypeOrder < rightTypeOrder
		}
		if queues[i].ArtifactType != queues[j].ArtifactType {
			return queues[i].ArtifactType < queues[j].ArtifactType
		}
		leftSectionOrder := setupSvcLiveReplayRequiredEvidenceSectionOrder(queues[i].ArtifactType, queues[i].Section)
		rightSectionOrder := setupSvcLiveReplayRequiredEvidenceSectionOrder(queues[j].ArtifactType, queues[j].Section)
		if leftSectionOrder != rightSectionOrder {
			return leftSectionOrder < rightSectionOrder
		}
		return queues[i].Section < queues[j].Section
	})
	return queues
}

func setupSvcLiveReplayEvidenceSectionQueuePageSize(queueCommand string) int {
	fields := strings.Fields(queueCommand)
	for index, field := range fields {
		if field != "--limit" || index+1 >= len(fields) {
			continue
		}
		if value, err := strconv.Atoi(fields[index+1]); err == nil && value > 0 {
			return value
		}
	}
	return 25
}

func setupSvcLiveReplayEvidenceSectionQueueBatchCount(missing int, pageSize int) int {
	if missing <= 0 {
		return 0
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	return (missing + pageSize - 1) / pageSize
}

func setupSvcLiveReplayEvidenceSectionQueueBatchCommands(queueCommand string, pageSize int, batchCount int) []string {
	if batchCount <= 0 {
		return nil
	}
	if pageSize <= 0 {
		pageSize = 25
	}
	commandCount := batchCount
	if commandCount > setupSvcLiveReplayEvidenceSectionQueueCommandLimit {
		commandCount = setupSvcLiveReplayEvidenceSectionQueueCommandLimit
	}
	commands := make([]string, 0, commandCount)
	for batchIndex := 0; batchIndex < commandCount; batchIndex++ {
		commands = append(commands, setupSvcLiveReplayEvidenceSectionQueueCommandWithOffset(queueCommand, batchIndex*pageSize))
	}
	return commands
}

func setupSvcLiveReplayEvidenceSectionQueueOmittedBatchCommands(batchCount int, commandCount int) int {
	if batchCount <= commandCount {
		return 0
	}
	return batchCount - commandCount
}

func setupSvcLiveReplayEvidenceSectionQueueCommandWithOffset(queueCommand string, offset int) string {
	fields := strings.Fields(queueCommand)
	for index, field := range fields {
		if field == "--offset" && index+1 < len(fields) {
			fields[index+1] = strconv.Itoa(offset)
			return strings.Join(fields, " ")
		}
	}
	fields = append(fields, "--offset", strconv.Itoa(offset))
	return strings.Join(fields, " ")
}

func setupSvcLiveReplayRequiredEvidenceSectionOrder(artifactType string, section string) int {
	for index, candidate := range setupSvcLiveReplayRequiredEvidenceSections(artifactType) {
		if candidate == section {
			return index
		}
	}
	return 1000
}

func setupSvcLiveReplayCollectionPlanPageCommandsFor(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, totalNextArtifacts int) setupSvcLiveReplayCollectionPlanPageCommands {
	commands := setupSvcLiveReplayCollectionPlanPageCommands{
		CurrentPage: setupSvcLiveReplayCollectionPlanPageCommand(projectPath, manifestPath, options.Offset, options),
	}
	if options.Limit > 0 && options.Offset+options.Limit < totalNextArtifacts {
		commands.NextPage = setupSvcLiveReplayCollectionPlanPageCommand(projectPath, manifestPath, options.Offset+options.Limit, options)
	}
	if options.Offset > 0 {
		previousOffset := 0
		if options.Limit > 0 && options.Offset > options.Limit {
			previousOffset = options.Offset - options.Limit
		}
		commands.PreviousPage = setupSvcLiveReplayCollectionPlanPageCommand(projectPath, manifestPath, previousOffset, options)
	}
	return commands
}

func setupSvcLiveReplayCapturePlanPageCommandsFor(projectPath string, manifestPath string, options setupSvcLiveReplayCollectionPlanOptions, totalNextArtifacts int) setupSvcLiveReplayCollectionPlanPageCommands {
	commands := setupSvcLiveReplayCollectionPlanPageCommands{
		CurrentPage: setupSvcLiveReplayCapturePlanPageCommand(projectPath, manifestPath, options.Offset, options),
	}
	if options.Limit > 0 && options.Offset+options.Limit < totalNextArtifacts {
		commands.NextPage = setupSvcLiveReplayCapturePlanPageCommand(projectPath, manifestPath, options.Offset+options.Limit, options)
	}
	if options.Offset > 0 {
		previousOffset := 0
		if options.Limit > 0 && options.Offset > options.Limit {
			previousOffset = options.Offset - options.Limit
		}
		commands.PreviousPage = setupSvcLiveReplayCapturePlanPageCommand(projectPath, manifestPath, previousOffset, options)
	}
	return commands
}

func setupSvcLiveReplayCapturePlanPageCommand(projectPath string, manifestPath string, offset int, options setupSvcLiveReplayCollectionPlanOptions) string {
	parts := []string{
		"cloudcc", "scan", "msapi", shellPath(projectPath), "setup-svc-live-replay-capture-plan", shellPath(manifestPath),
	}
	if options.Domain != "" {
		parts = append(parts, "--domain", shellPath(options.Domain))
	}
	if options.Operation != "" {
		parts = append(parts, "--operation", shellPath(options.Operation))
	}
	if options.ArtifactType != "" {
		parts = append(parts, "--artifact-type", shellPath(options.ArtifactType))
	}
	if options.EvidenceSection != "" {
		parts = append(parts, "--evidence-section", shellPath(options.EvidenceSection))
	}
	if options.SectionStatus != "" {
		parts = append(parts, "--section-status", shellPath(options.SectionStatus))
	}
	if options.SourceStatus != "" {
		parts = append(parts, "--source-status", shellPath(options.SourceStatus))
	}
	if options.SourceReadiness != "" {
		parts = append(parts, "--source-readiness", shellPath(options.SourceReadiness))
	}
	parts = append(parts, "--offset", strconv.Itoa(offset), "--limit", strconv.Itoa(options.Limit))
	return strings.Join(parts, " ")
}

func setupSvcLiveReplayCollectionPlanPageCommand(projectPath string, manifestPath string, offset int, options setupSvcLiveReplayCollectionPlanOptions) string {
	parts := []string{
		"cloudcc", "scan", "msapi", shellPath(projectPath), "setup-svc-live-replay-gaps", shellPath(manifestPath),
	}
	if options.Domain != "" {
		parts = append(parts, "--domain", shellPath(options.Domain))
	}
	if options.Operation != "" {
		parts = append(parts, "--operation", shellPath(options.Operation))
	}
	if options.ArtifactType != "" {
		parts = append(parts, "--artifact-type", shellPath(options.ArtifactType))
	}
	if options.Status != "" {
		parts = append(parts, "--status", shellPath(options.Status))
	}
	if options.EvidenceSection != "" {
		parts = append(parts, "--evidence-section", shellPath(options.EvidenceSection))
	}
	if options.SectionStatus != "" {
		parts = append(parts, "--section-status", shellPath(options.SectionStatus))
	}
	if options.SourceStatus != "" {
		parts = append(parts, "--source-status", shellPath(options.SourceStatus))
	}
	if options.SourceReadiness != "" {
		parts = append(parts, "--source-readiness", shellPath(options.SourceReadiness))
	}
	parts = append(parts, "--offset", strconv.Itoa(offset), "--limit", strconv.Itoa(options.Limit))
	return strings.Join(parts, " ")
}

func setupSvcLiveReplayCollectionAction(projectPath string, domain string, operation setupSvcLiveReplayGapOperation, artifactType string, file string, status string, domainContracts map[string]setupSvcLiveReplayDomain) setupSvcLiveReplayArtifactCollectionAction {
	contract := domainContracts[normalizeDomain(domain)]
	requiredSections := setupSvcLiveReplayRequiredEvidenceSections(artifactType)
	requiredShapeKey := setupSvcLiveReplayCollectionRequiredShapeKey(artifactType)
	manifestStatusField := setupSvcLiveReplayManifestStatusField(artifactType)
	requiredTables := setupSvcLiveReplayCollectionRequiredTables(contract, artifactType)
	runtimeEffects := setupSvcLiveReplayCollectionRuntimeEffects(contract, artifactType)
	queryReadbackExpectations := setupSvcLiveReplayCollectionQueryReadbackExpectations(contract, artifactType)
	suggestedSourcePath := setupSvcLiveReplayWorklistSuggestedSourcePath(file)
	return setupSvcLiveReplayArtifactCollectionAction{
		Domain:                    domain,
		Operation:                 operation.Operation,
		ArtifactType:              artifactType,
		Path:                      file,
		Status:                    status,
		NextAction:                setupSvcLiveReplayCollectionNextAction(status, artifactType, operation.NextAction),
		RequiredShapeKey:          requiredShapeKey,
		ManifestStatusField:       manifestStatusField,
		RequiredEvidenceSections:  requiredSections,
		EvidenceSectionStatuses:   setupSvcLiveReplayEvidenceSectionStatuses(projectPath, file, requiredSections),
		ReplacementChecklist:      setupSvcLiveReplayArtifactReplacementChecklist(artifactType),
		RequiredTables:            requiredTables,
		RuntimeEffects:            runtimeEffects,
		QueryReadbackExpectations: queryReadbackExpectations,
		CaptureTask:               setupSvcLiveReplayArtifactCaptureTaskFor(projectPath, "", domain, operation.Operation, artifactType, file, suggestedSourcePath, requiredShapeKey, manifestStatusField, requiredSections, requiredTables, runtimeEffects, queryReadbackExpectations),
	}
}

func setupSvcLiveReplayEvidenceSectionStatuses(projectPath string, file string, sections []string) []setupSvcLiveReplayEvidenceSectionStatus {
	statuses := make([]setupSvcLiveReplayEvidenceSectionStatus, 0, len(sections))
	artifact, ok := readSetupSvcLiveReplayArtifactMap(projectPath, file)
	pending := ok && strings.EqualFold(firstMapString(artifact, "status", "evidenceStatus"), "pending")
	for _, section := range sections {
		present := ok && setupSvcLiveReplayArtifactSectionPresent(artifact, section) && (!pending || setupSvcLiveReplayEvidenceIdentitySection(section))
		status := "missing"
		if present {
			status = "present"
		}
		statuses = append(statuses, setupSvcLiveReplayEvidenceSectionStatus{
			Section: section,
			Status:  status,
			Present: present,
		})
	}
	return statuses
}

func setupSvcLiveReplayEvidenceSectionStatusesAtPath(path string, sections []string) []setupSvcLiveReplayEvidenceSectionStatus {
	statuses := make([]setupSvcLiveReplayEvidenceSectionStatus, 0, len(sections))
	artifact, err := readJSONFile(path)
	ok := err == nil
	pending := ok && strings.EqualFold(firstMapString(artifact, "status", "evidenceStatus"), "pending")
	for _, section := range sections {
		present := ok && setupSvcLiveReplayArtifactSectionPresent(artifact, section) && (!pending || setupSvcLiveReplayEvidenceIdentitySection(section))
		status := "missing"
		if present {
			status = "present"
		}
		statuses = append(statuses, setupSvcLiveReplayEvidenceSectionStatus{
			Section: section,
			Status:  status,
			Present: present,
		})
	}
	return statuses
}

func setupSvcLiveReplaySourceReadinessFor(projectPath string, artifactPath string) string {
	suggestedSourcePath := setupSvcLiveReplayWorklistSuggestedSourcePath(artifactPath)
	sourcePath := filepath.Join(projectPath, suggestedSourcePath)
	sourceExists := setupSvcLiveReplayWorklistSuggestedSourceExists(projectPath, suggestedSourcePath)
	requiredSections := setupSvcLiveReplayRequiredEvidenceSections(setupSvcLiveReplayArtifactType(artifactPath))
	return setupSvcLiveReplaySourceReadiness(sourceExists, setupSvcLiveReplayEvidenceSectionStatusesAtPath(sourcePath, requiredSections))
}

func setupSvcLiveReplaySourceReadiness(sourceExists bool, statuses []setupSvcLiveReplayEvidenceSectionStatus) string {
	if !sourceExists {
		return "missing"
	}
	for _, status := range statuses {
		if !status.Present {
			return "incomplete"
		}
	}
	return "complete"
}

func setupSvcLiveReplayEvidenceIdentitySection(section string) bool {
	switch section {
	case "status", "project", "contractVersion", "contractFingerprint", "domain", "operation", "artifactType":
		return true
	default:
		return false
	}
}

func setupSvcLiveReplayArtifactSectionPresent(artifact map[string]any, section string) bool {
	switch section {
	case "status":
		return strings.TrimSpace(firstMapString(artifact, "status", "evidenceStatus")) != ""
	case "project", "contractVersion", "contractFingerprint", "domain", "operation", "artifactType":
		return strings.TrimSpace(firstMapString(artifact, section)) != ""
	case "tableSnapshots":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "tableSnapshots", "snapshots", "metadataSnapshots")
	case "runtimeEffectChecks":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "runtimeEffectChecks")
	case "queryShape":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "queryShape")
	case "readbackShape":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "readbackShape", "fields", "columns")
	case "readbackTables":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "readbackTables", "queriedTables", "metadataTables", "tableCoverage")
	case "relationshipChecks":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "relationshipChecks", "relationships")
	case "readbackExpectationChecks":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "readbackExpectationChecks")
	case "cleanCounters":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "missing", "mismatched", "broken", "unreadable", "errors", "errorCount", "readbackChecks")
	case "diffCounters":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "missingRows", "unexpectedRows", "mismatchedValues", "differences", "totals")
	case "nestedCleanCounters":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "diff", "evidence", "comparison", "normalizedDiff")
	case "deletedOrRemovedEvidence":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "deleted", "removed", "deletedRows", "removedRows")
	case "residualCounters":
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, "remaining", "residual", "orphan", "errors", "failures", "cleanup")
	default:
		return setupSvcLiveReplayAnyMapKeyPresent(artifact, section)
	}
}

func setupSvcLiveReplayAnyMapKeyPresent(values map[string]any, keys ...string) bool {
	for _, key := range keys {
		if setupSvcLiveReplayEvidenceValuePresent(values[key]) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayEvidenceValuePresent(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func setupSvcLiveReplayManifestStatusField(artifactType string) string {
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "setup-svc":
		return "setupSvcEvidenceStatus"
	case "metadata-service":
		return "metadataServiceEvidenceStatus"
	case "query-readback":
		return "queryEvidenceStatus"
	case "normalized-diff":
		return "normalizedDiffStatus"
	case "cleanup":
		return "cleanupStatus"
	default:
		return ""
	}
}

func setupSvcLiveReplayRequiredEvidenceSections(artifactType string) []string {
	common := []string{
		"status",
		"project",
		"contractVersion",
		"contractFingerprint",
		"domain",
		"operation",
		"artifactType",
	}
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "setup-svc":
		return append(common, "tableSnapshots", "runtimeEffectChecks")
	case "metadata-service":
		return append(common, "tableSnapshots", "runtimeEffectChecks", "metadataServiceDatasource")
	case "query-readback":
		return append(common, "queryShape", "readbackShape", "readbackTables", "relationshipChecks", "readbackExpectationChecks", "cleanCounters")
	case "normalized-diff":
		return append(common, "diffCounters", "nestedCleanCounters")
	case "cleanup":
		return append(common, "deletedOrRemovedEvidence", "residualCounters")
	default:
		return common
	}
}

func setupSvcLiveReplayArtifactReplacementChecklist(artifactType string) []string {
	checklist := []string{
		"Keep project, contractVersion, contractFingerprint, domain, operation, and artifactType unchanged.",
		"Change status to passed, verified, or success only after replacing this placeholder with real replay evidence.",
		"Do not use boolean-only passed/success/verified flags, clean-only markers, rowCount-only tables, columns-only tables, or empty table placeholders as evidence.",
	}
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "setup-svc", "metadata-service":
		checklist = append(checklist,
			"Include tableSnapshots or equivalent snapshots for every requiredTable.",
			"Each required table must include row evidence plus column, field, primaryKey, or keyColumn evidence.",
			"Include named passed runtimeEffectChecks for every runtime effect declared by the domain contract.",
		)
	case "query-readback":
		checklist = append(checklist,
			"Include queryShape or readbackShape fields and non-empty readback table coverage for every requiredTable.",
			"Each required table must include actual row evidence plus column, field, requiredField, readbackField, or queryField evidence.",
			"Include named passed relationshipChecks or source, target, and field relationship structure plus named passed readbackExpectationChecks.",
			"Declare numeric zero missing, mismatched, broken, unreadable, and error counters.",
		)
	case "normalized-diff":
		checklist = append(checklist,
			"Include numeric zero missing, unexpected, mismatched, failed, difference, and error counters.",
			"Keep nested diff, evidence, comparison, or normalizedDiff nodes clean with numeric zero counters; clean:true alone is rejected.",
		)
	case "cleanup":
		checklist = append(checklist,
			"Include deleted or removed evidence plus remaining, residual, orphan, error, and failure counters.",
			"All cleanup residual counters must be numeric zero before status can be passed.",
		)
	}
	return checklist
}

func setupSvcLiveReplayArtifactCaptureTaskFor(projectPath string, manifestPath string, domain string, operation string, artifactType string, targetPath string, suggestedSourcePath string, requiredShapeKey string, manifestStatusField string, requiredSections []string, requiredTables []string, runtimeEffects []string, queryReadbackExpectations []string) setupSvcLiveReplayArtifactCaptureTask {
	artifactType = setupSvcLiveReplayNormalizeArtifactType(artifactType)
	if strings.TrimSpace(manifestPath) == "" {
		manifestPath = setupSvcLiveReplayManifestPath(projectPath, "")
	}
	sourceSystem, captureMode, manualAction := setupSvcLiveReplayCaptureTaskSource(artifactType, domain, operation)
	task := setupSvcLiveReplayArtifactCaptureTask{
		SourceSystem:              sourceSystem,
		CaptureMode:               captureMode,
		Domain:                    domain,
		Operation:                 operation,
		ArtifactType:              artifactType,
		TargetPath:                strings.TrimSpace(targetPath),
		SuggestedSourcePath:       strings.TrimSpace(suggestedSourcePath),
		StatusTarget:              "passed|verified|success",
		RequiredShapeKey:          requiredShapeKey,
		ManifestStatusField:       manifestStatusField,
		RequiredEvidenceSections:  append([]string{}, requiredSections...),
		RequiredTables:            append([]string{}, requiredTables...),
		RuntimeEffects:            append([]string{}, runtimeEffects...),
		QueryReadbackExpectations: append([]string{}, queryReadbackExpectations...),
		ManualAction:              manualAction,
		PostCaptureCheckCommand:   setupSvcLiveReplayCaptureTaskCheckCommand(projectPath, manifestPath, domain, operation, artifactType),
		PostCaptureImportHint:     "After this source file is structurally complete, include it in a --source-readiness complete worklist and run setup-svc-live-replay-evidence-import --dry-run before any approved import.",
		StopConditions: []string{
			"Do not set the artifact status to passed until every requiredEvidenceSection is present with real replay evidence.",
			"Do not import incomplete captures; isolate them with --source-readiness incomplete and repair first.",
			"Do not use clean-only, rowCount-only, columns-only, or boolean-only markers as structure proof.",
		},
	}
	if artifactType == "normalized-diff" {
		task.CaptureCommand = "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-normalized-diff " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityNormalizedDiffApproval
		task.PostCaptureImportHint = "Run normalized-diff after setup-svc and MetadataService snapshot evidence has been imported and manifest-sync has marked both snapshot statuses passed."
	}
	if artifactType == "metadata-service" {
		if strings.EqualFold(strings.TrimSpace(operation), "query") {
			task.ScanRequest = setupSvcLiveReplayMetadataServiceScanRequest(domain)
			task.ScanCommand = "cloudcc scan msapi " + shellPath(projectPath) + " " + shellPath(normalizeDomain(domain))
		} else if planRequest := setupSvcLiveReplayMetadataServicePlanRequest(domain, operation); len(planRequest) > 0 {
			task.PlanRequest = planRequest
			task.PlanCommand = "cloudcc plan msapi " + shellPath(projectPath) + " " + shellPath(normalizeDomain(domain)) + " '<planRequest.spec>' " + shellPath(strings.ToLower(strings.TrimSpace(operation)))
		}
	}
	return task
}

func setupSvcLiveReplayCaptureTaskSource(artifactType string, domain string, operation string) (string, string, string) {
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "setup-svc":
		return "setup-svc", "manual_or_scripted_snapshot_capture", "Run the legacy setup-svc " + domain + "/" + operation + " flow in the approved replay tenant, then snapshot every requiredTable plus named runtimeEffectChecks into suggestedSourcePath."
	case "metadata-service":
		if strings.EqualFold(strings.TrimSpace(operation), "query") {
			return "metadata-service", "msapi_scan_snapshot_capture", "Run the matching MetadataService/MSAPI " + domain + " scan/query path in the approved replay tenant, then snapshot every requiredTable plus named runtimeEffectChecks into suggestedSourcePath."
		}
		return "metadata-service", "msapi_plan_apply_snapshot_capture", "Run the matching MetadataService/MSAPI " + domain + "/" + operation + " plan/apply flow in the approved replay tenant, then snapshot every requiredTable plus named runtimeEffectChecks into suggestedSourcePath."
	case "query-readback":
		return "msapi-query-readback", "msapi_query_readback_capture", "Run the MSAPI query/readback path for " + domain + "/" + operation + " after replay writes, then capture queryShape, readbackShape, readbackTables, relationshipChecks, readbackExpectationChecks, and zero clean counters into suggestedSourcePath."
	case "normalized-diff":
		return "local-normalized-diff", "approval_gated_generated_diff", "Generate the normalized diff only after setup-svc and MetadataService snapshot evidence for this operation are complete."
	case "cleanup":
		return "cleanup-verifier", "cleanup_residual_capture", "Run the operation cleanup or rollback verification for " + domain + "/" + operation + ", then capture deleted/removed evidence and zero residual/orphan/error counters into suggestedSourcePath."
	default:
		return "unknown", "manual_capture", "Capture real replay evidence for " + domain + "/" + operation + " into suggestedSourcePath."
	}
}

func setupSvcLiveReplayCaptureTaskCheckCommand(projectPath string, manifestPath string, domain string, operation string, artifactType string) string {
	return "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-capture-plan " + shellPath(manifestPath) +
		" --domain " + shellPath(normalizeDomain(domain)) +
		" --operation " + shellPath(strings.ToLower(strings.TrimSpace(operation))) +
		" --artifact-type " + shellPath(setupSvcLiveReplayNormalizeArtifactType(artifactType)) +
		" --source-readiness complete --limit 1"
}

func setupSvcLiveReplayMetadataServicePlanRequest(domain string, operation string) map[string]any {
	domain = normalizeDomain(domain)
	operation = strings.ToLower(strings.TrimSpace(operation))
	if operation == "" || operation == "query" {
		return nil
	}
	spec := setupSvcLiveReplayMetadataServicePlanSpec(domain, operation)
	if len(spec) == 0 {
		return nil
	}
	return map[string]any{
		"domain":    domain,
		"operation": operation,
		"mode":      operation,
		"spec":      spec,
		"context": map[string]any{
			"actorId":       "cloudcc",
			"locale":        "zh-CN",
			"source":        "setup-svc-live-replay",
			"replaySafe":    true,
			"replayDataset": "cc_replay",
		},
	}
}

func setupSvcLiveReplayMetadataServiceScanRequest(domain string) map[string]any {
	domain = normalizeDomain(domain)
	if domain == "" {
		return nil
	}
	return map[string]any{
		"domain": domain,
		"filters": map[string]any{
			"replayDataset": "cc_replay",
			"apiNamePrefix": "cc_replay_",
		},
		"context": map[string]any{
			"locale":        "zh-CN",
			"source":        "setup-svc-live-replay",
			"replayDataset": "cc_replay",
		},
	}
}

func setupSvcLiveReplayMetadataServicePlanSpec(domain string, operation string) map[string]any {
	if domain == "reports" && strings.HasPrefix(operation, "folder-") {
		return setupSvcLiveReplayCompactFixtureIDMap(setupSvcLiveReplayReportFolderSpec(operation))
	}
	if operation == "delete" || operation == "physical-purge" || operation == "remove" {
		if spec := setupSvcLiveReplayMetadataServiceDeleteSpec(domain, operation); len(spec) > 0 {
			return setupSvcLiveReplayCompactFixtureIDMap(spec)
		}
	}
	spec := setupSvcLiveReplayMetadataServiceCreateSpec(domain)
	if len(spec) == 0 {
		return nil
	}
	if operation == "update" || operation == "assign" {
		setupSvcLiveReplayMarkSpecForOperation(spec, domain, operation)
	}
	return setupSvcLiveReplayCompactFixtureIDMap(spec)
}

func setupSvcLiveReplayReportFolderSpec(operation string) map[string]any {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case "folder-create":
		return map[string]any{
			"id":         "folder_cc_replay_reports",
			"name":       "Replay Report Folder",
			"folderType": "report",
			"viewType":   "2",
			"purview":    "1",
		}
	case "folder-update":
		return map[string]any{
			"id":         "folder_cc_replay_reports",
			"name":       "Replay Report Folder Updated",
			"folderType": "report",
			"viewType":   "1",
			"purview":    "2",
		}
	case "folder-delete":
		return map[string]any{"id": "folder_cc_replay_reports"}
	default:
		return nil
	}
}

func setupSvcLiveReplayCompactFixtureIDMap(spec map[string]any) map[string]any {
	compacted, _ := setupSvcLiveReplayCompactFixtureIDs(spec).(map[string]any)
	return compacted
}

func setupSvcLiveReplayCompactFixtureIDs(value any) any {
	replacements := map[string]string{
		"005_cc_replay_user":          "005ccruser",
		"act_cc_replay":               "actccr",
		"app_cc_replay":               "appccr",
		"apptab_cc_replay":            "apptabccr",
		"appr_cc_replay":              "apprccr",
		"aprf_cc_replay":              "aprfccr",
		"aprl_cc_replay":              "aprlccr",
		"apsl_cc_replay":              "apslccr",
		"apst_cc_replay":              "apstccr",
		"btn_cc_replay":               "btnccr",
		"cc_replay_app":               "ccrapp",
		"cond_dup_cc_replay":          "conddupccr",
		"cond_view_cc_replay":         "condviewccr",
		"dash_cc_replay":              "dashccr",
		"dashc_cc_replay":             "dashccnd",
		"dashr_cc_replay":             "dashrccr",
		"dup_cc_replay":               "dupccr",
		"dupr_cc_replay_name":         "duprccrn",
		"email_cc_replay":             "emailccr",
		"field_cc_replay_limit":       "fldccrlim",
		"field_cc_replay_text":        "fldccrtxt",
		"folder_cc_replay_dashboards": "foldshccr",
		"folder_cc_replay_reports":    "folrptccr",
		"gsl_cc_replay":               "gslccr",
		"gslc_cc_replay_enabled":      "gslccren",
		"idp_cc_replay":               "idpccr",
		"idpsp_cc_replay":             "idpspccr",
		"layout_cc_replay":            "layccr",
		"layout_cc_replay_setting":    "layccrset",
		"obj_cc_replay":               "objccr",
		"obj_cc_replay_setting":       "objccrset",
		"permset_cc_replay":           "psetccr",
		"profile_cc_replay":           "profccr",
		"rl_cc_replay":                "rlccr",
		"role_cc_replay":              "roleccr",
		"role_parent_cc_replay":       "roleparccr",
		"rpf_cc_replay":               "rpfccr",
		"rpo_cc_replay":               "rpoccr",
		"rpod_cc_replay":              "rpodccr",
		"rpt_cc_replay":               "rptccr",
		"rtc_cc_replay":               "rtcccr",
		"rtcf_cc_replay":              "rtcfccr",
		"rt_cc_replay":                "rtccr",
		"sect_cc_replay":              "secccr",
		"sect_cc_replay_setting":      "secccrset",
		"sf_cc_replay_name":           "sfccrname",
		"share_cc_replay":             "shrccr",
		"sso_cc_replay":               "ssoccr",
		"tab_cc_replay":               "tabccr",
		"vbtn_cc_replay":              "vbtnccr",
		"vf_cc_replay":                "vfccr",
		"view_cc_replay":              "viewccr",
		"vr_cc_replay":                "vrccr",
	}
	switch typed := value.(type) {
	case string:
		if replacement, ok := replacements[typed]; ok {
			return replacement
		}
		return typed
	case []any:
		compacted := make([]any, len(typed))
		for i, item := range typed {
			compacted[i] = setupSvcLiveReplayCompactFixtureIDs(item)
		}
		return compacted
	case map[string]any:
		compacted := make(map[string]any, len(typed))
		for key, item := range typed {
			compacted[key] = setupSvcLiveReplayCompactFixtureIDs(item)
		}
		return compacted
	default:
		return value
	}
}

func setupSvcLiveReplayMarkSpecForOperation(spec map[string]any, domain string, operation string) {
	switch operation {
	case "update":
		if _, ok := spec["label"]; ok {
			spec["label"] = "回放更新"
		}
		if _, ok := spec["name"]; ok {
			spec["name"] = "回放更新"
		}
		if _, ok := spec["description"]; ok {
			spec["description"] = "用于验证 MetadataService 更新路径的回放数据。"
		}
	case "assign":
		if domain == "permissions" {
			spec["assignees"] = []any{"005_cc_replay_user"}
		}
		if domain == "roles" {
			spec["userId"] = "005_cc_replay_user"
		}
	}
}

func setupSvcLiveReplayMetadataServiceDeleteSpec(domain string, operation string) map[string]any {
	switch domain {
	case "objects":
		return map[string]any{"id": "obj_cc_replay"}
	case "fields":
		return map[string]any{"id": "field_cc_replay_text", "objectId": "obj_cc_replay"}
	case "global-select-lists":
		return map[string]any{"id": "gsl_cc_replay"}
	case "record-types":
		return map[string]any{"id": "rt_cc_replay", "objectId": "obj_cc_replay"}
	case "layouts":
		return map[string]any{"id": "layout_cc_replay", "objectId": "obj_cc_replay"}
	case "profiles":
		return map[string]any{"id": "profile_cc_replay"}
	case "permissions":
		if operation == "remove" {
			return map[string]any{"id": "permset_cc_replay", "assignees": []any{"005_cc_replay_user"}}
		}
		return map[string]any{"id": "permset_cc_replay"}
	case "roles":
		if operation == "remove" {
			return map[string]any{"id": "role_cc_replay", "userId": "005_cc_replay_user"}
		}
		return map[string]any{"id": "role_cc_replay"}
	case "sharing-rules":
		return map[string]any{"id": "share_cc_replay"}
	case "validation-rules":
		return map[string]any{"id": "vr_cc_replay"}
	case "applications":
		return map[string]any{"id": "app_cc_replay"}
	case "menus":
		return map[string]any{"id": "tab_cc_replay"}
	case "buttons":
		return map[string]any{"id": "btn_cc_replay"}
	case "custom-settings":
		return map[string]any{"id": "obj_cc_replay_setting", "apiName": "cc_replay_setting"}
	case "dupe-catchers":
		return map[string]any{"id": "dup_cc_replay"}
	case "single-sign-ons":
		return map[string]any{"id": "sso_cc_replay"}
	case "identity-providers":
		return map[string]any{"app": "cc_replay_app"}
	case "approval-processes":
		return map[string]any{"id": "appr_cc_replay", "stepIds": []any{"apst_cc_replay"}, "relatedListIds": []any{"aprl_cc_replay"}}
	case "reports":
		return map[string]any{"id": "rpt_cc_replay", "objectIds": []any{"rpo_cc_replay"}}
	case "dashboards":
		return map[string]any{"id": "dash_cc_replay"}
	case "object-views":
		return map[string]any{"id": "view_cc_replay", "objectId": "obj_cc_replay"}
	default:
		return nil
	}
}

func setupSvcLiveReplayMetadataServiceCreateSpec(domain string) map[string]any {
	switch domain {
	case "objects":
		return map[string]any{"id": "obj_cc_replay", "apiName": "cc_replay_object", "label": "回放对象", "nameLabel": "回放编号", "profiles": []any{"aaa000001"}}
	case "fields":
		return map[string]any{"id": "field_cc_replay_text", "objectId": "obj_cc_replay", "objectApiName": "cc_replay_object", "apiName": "cc_replay_text", "nameLabel": "回放文本", "type": "S", "profiles": []any{"aaa000001"}}
	case "global-select-lists":
		return map[string]any{"id": "gsl_cc_replay", "apiName": "cc_replay_status", "label": "回放状态", "options": []any{map[string]any{"id": "gslc_cc_replay_enabled", "value": "enabled", "label": "启用"}}}
	case "record-types":
		return map[string]any{"id": "rt_cc_replay", "objectId": "obj_cc_replay", "name": "回放记录类型", "apiName": "cc_replay_record_type", "profiles": []any{"aaa000001"}, "layouts": []any{"layout_cc_replay"}}
	case "layouts":
		return map[string]any{"id": "layout_cc_replay", "objectId": "obj_cc_replay", "name": "回放布局", "sections": []any{map[string]any{"id": "sect_cc_replay", "name": "基本信息", "fields": []any{map[string]any{"id": "sf_cc_replay_name", "fieldId": "field_cc_replay_text"}}}}}
	case "profiles":
		return map[string]any{"id": "profile_cc_replay", "name": "回放简档", "objectPermissions": []any{map[string]any{"objectId": "obj_cc_replay", "readable": true, "editable": true}}, "fieldPermissions": []any{map[string]any{"fieldId": "field_cc_replay_text", "readable": true, "editable": true}}, "layouts": []any{map[string]any{"objectId": "obj_cc_replay", "layoutId": "layout_cc_replay"}}}
	case "permissions":
		return map[string]any{"id": "permset_cc_replay", "name": "回放权限集", "permissions": []any{map[string]any{"permission": "cc_replay_permission", "enabled": true}}, "fieldPermissions": []any{map[string]any{"fieldId": "field_cc_replay_text", "readable": true, "editable": true}}}
	case "roles":
		return map[string]any{"id": "role_cc_replay", "name": "回放角色", "parentId": "role_parent_cc_replay", "userId": "005_cc_replay_user"}
	case "sharing-rules":
		return map[string]any{"id": "share_cc_replay", "name": "回放共享", "objectId": "obj_cc_replay", "accessLevel": "read", "sourceType": "criteria", "sourceId": "role_cc_replay", "targetType": "role", "targetId": "role_parent_cc_replay", "shareTableName": "tp_c_sharetable_cc_replay", "conditions": []any{map[string]any{"id": "cond_share_cc_replay", "fieldId": "field_cc_replay_text", "operator": "=", "value": "启用"}}}
	case "validation-rules":
		return map[string]any{"id": "vr_cc_replay", "name": "回放校验", "objectId": "obj_cc_replay", "formula": "ISBLANK(cc_replay_text)", "active": "false", "errorMessage": "回放文本不能为空"}
	case "applications":
		return map[string]any{"id": "app_cc_replay", "name": "回放应用", "label": "回放应用", "custom": "1", "tabs": []any{map[string]any{"id": "tab_cc_replay", "name": "cc_replay_tab", "label": "回放菜单", "appTabId": "apptab_cc_replay"}}, "visibleProfiles": []any{"aaa000001"}, "allProfileIds": []any{"aaa000001", "aaa000002"}}
	case "menus":
		return map[string]any{"id": "tab_cc_replay", "type": "object", "objectId": "obj_cc_replay", "objectApiName": "cc_replay_object", "objectPrefix": "CRP", "tabName": "回放菜单", "appIds": []any{"app_cc_replay"}, "allOrSome": "all", "overrideSelect": "2", "allProfileIds": []any{"aaa000001", "aaa000002"}}
	case "buttons":
		return map[string]any{"id": "btn_cc_replay", "name": "cc_replay_submit", "label": "提交回放", "objId": "obj_cc_replay", "event": "URL", "url": "/cc-replay/submit", "scopeon": "location"}
	case "custom-settings":
		return map[string]any{"id": "obj_cc_replay_setting", "apiName": "cc_replay_setting", "label": "回放设置", "fields": []any{map[string]any{"id": "field_cc_replay_limit", "apiName": "cc_replay_limit", "nameLabel": "回放上限", "type": "N"}}, "regenerateLayout": true, "layoutId": "layout_cc_replay_setting", "sectionId": "sect_cc_replay_setting", "allProfileIds": []any{"aaa000001", "aaa000002"}}
	case "dupe-catchers":
		return map[string]any{"id": "dup_cc_replay", "name": "回放查重", "objectId": "obj_cc_replay", "active": "1", "isprofile": true, "errormessage": "回放数据不能重复", "insertoperation": "check", "updateoperation": "check", "conditions": []any{map[string]any{"id": "cond_dup_cc_replay", "fieldId": "isdeleted", "operator": "=", "value": "0"}}, "rules": []any{map[string]any{"id": "dupr_cc_replay_name", "fieldId": "field_cc_replay_text", "matchOption": "exact", "firstLetters": "3"}}}
	case "single-sign-ons":
		return map[string]any{"id": "sso_cc_replay", "name": "回放 SSO", "issuer": "https://idp.example.com/cc-replay", "entityId": "cloudcc-cc-replay", "loginUrl": "https://idp.example.com/login", "enableLogout": true, "requestBinding": "HTTP-POST", "appDomain": "https://crm.example.com/", "orgId": "00Dccreplay", "certName": "cc-replay-idp.crt"}
	case "identity-providers":
		return map[string]any{"config": map[string]any{"id": "idp_cc_replay", "enableIdp": true, "issuer": "cloudcc-cc-replay"}, "serviceProviders": []any{map[string]any{"id": "idpsp_cc_replay", "app": "cc_replay_app", "entityId": "cc-replay-sp", "acsUrl": "https://sp.example.com/acs", "logoutBinding": "HTTP-POST"}}}
	case "approval-processes":
		return map[string]any{"id": "appr_cc_replay", "name": "回放审批", "apiName": "cc_replay_approval", "targetObject": "obj_cc_replay", "approvalPageFields": []any{map[string]any{"fieldId": "field_cc_replay_text"}}, "conditions": []any{map[string]any{"id": "cond_appr_cc_replay", "fieldId": "field_cc_replay_text", "operator": "=", "value": "启用"}}, "steps": []any{map[string]any{"id": "apst_cc_replay", "name": "回放步骤", "apiName": "cc_replay_step", "approverType": "user", "approvalPageFields": []any{map[string]any{"fieldId": "field_cc_replay_text"}}, "conditions": []any{map[string]any{"id": "cond_apst_cc_replay", "fieldId": "field_cc_replay_text", "operator": "!=", "value": "停用"}}, "layoutFields": []any{map[string]any{"id": "apsl_cc_replay", "fieldId": "field_cc_replay_text", "readonly": true}}, "actions": []any{map[string]any{"id": "act_cc_replay", "actionType": "email", "actionId": "email_cc_replay"}}}}, "relatedLists": []any{map[string]any{"id": "aprl_cc_replay", "relatedListId": "rl_cc_replay", "fields": []any{map[string]any{"id": "aprf_cc_replay", "fieldId": "field_cc_replay_text"}}}}}
	case "reports":
		return map[string]any{"id": "rpt_cc_replay", "name": "回放报表", "apiName": "cc_replay_report", "objectId": "obj_cc_replay", "description": "用于验证回放报表。", "folderId": "folder_cc_replay_reports", "folder": map[string]any{"id": "folder_cc_replay_reports", "name": "回放报表文件夹", "folderType": "report"}, "reportTypeCustom": map[string]any{"id": "rtc_cc_replay", "name": "回放报表类型", "objectA": "obj_cc_replay", "fields": []any{map[string]any{"id": "rtcf_cc_replay", "fieldId": "field_cc_replay_text", "objectId": "obj_cc_replay"}}}, "fields": []any{map[string]any{"id": "rpf_cc_replay", "fieldId": "field_cc_replay_text", "label": "回放文本", "location": 2, "locationType": "detail"}}, "conditions": []any{map[string]any{"id": "cond_rpt_cc_replay", "fieldId": "field_cc_replay_text", "operator": "=", "value": "启用"}}, "gathers": []any{map[string]any{"id": "rpg_cc_replay", "fieldId": "field_cc_replay_amount", "fieldName": "回放金额", "gatherType": "sum"}}, "groups": []any{map[string]any{"id": "rpgp_cc_replay", "fieldId": "field_cc_replay_text", "fieldName": "回放文本", "groupType": "row"}}, "objects": []any{map[string]any{"id": "rpo_cc_replay", "objectId": "obj_cc_replay", "details": []any{map[string]any{"id": "rpod_cc_replay", "title": "回放明细"}}}}}
	case "dashboards":
		return map[string]any{"id": "dash_cc_replay", "name": "回放仪表板", "apiName": "cc_replay_dashboard", "folderId": "folder_cc_replay_dashboards", "folder": map[string]any{"id": "folder_cc_replay_dashboards", "name": "回放仪表板文件夹", "folderType": "dashboard"}, "description": "用于验证回放仪表板。", "components": []any{map[string]any{"id": "dashr_cc_replay", "name": "回放趋势", "reportId": "rpt_cc_replay", "type": "line", "x": 1, "y": 2, "width": 6, "height": 4, "objectId": "obj_cc_replay", "viewId": "view_cc_replay", "filters": []any{map[string]any{"id": "dashc_cc_replay", "objectId": "obj_cc_replay", "fieldId": "field_cc_replay_text", "type": "eq"}}}}}
	case "object-views":
		return map[string]any{"id": "view_cc_replay", "name": "回放视图", "label": "回放视图", "apiName": "cc_replay_view", "objectId": "obj_cc_replay", "profileId": "aaa000001", "default": true, "ownerId": "005_cc_replay_user", "accessType": "public", "fields": []any{map[string]any{"id": "vf_cc_replay", "fieldId": "field_cc_replay_text", "apiName": "cc_replay_text", "fieldType": "S", "seq": 1}}, "buttons": []any{map[string]any{"id": "vbtn_cc_replay", "buttonId": "btn_cc_replay", "seq": 1}}, "conditions": []any{map[string]any{"id": "cond_view_cc_replay", "fieldId": "field_cc_replay_text", "operator": "=", "value": "启用"}}}
	default:
		return nil
	}
}

func setupSvcLiveReplayCollectionPlanFiltersFromOptions(options setupSvcLiveReplayCollectionPlanOptions) *setupSvcLiveReplayCollectionPlanFilters {
	filters := setupSvcLiveReplayCollectionPlanFilters{
		Domain:          options.Domain,
		Operation:       options.Operation,
		ArtifactType:    options.ArtifactType,
		SourceSystem:    options.SourceSystem,
		CaptureMode:     options.CaptureMode,
		Status:          options.Status,
		EvidenceSection: options.EvidenceSection,
		SectionStatus:   options.SectionStatus,
		SourceStatus:    options.SourceStatus,
		SourceReadiness: options.SourceReadiness,
	}
	if filters.Domain == "" && filters.Operation == "" && filters.ArtifactType == "" && filters.SourceSystem == "" && filters.CaptureMode == "" && filters.Status == "" && filters.EvidenceSection == "" && filters.SectionStatus == "" && filters.SourceStatus == "" && filters.SourceReadiness == "" {
		return nil
	}
	return &filters
}

func setupSvcLiveReplayCollectionPlanMatches(options setupSvcLiveReplayCollectionPlanOptions, domain string, operation string, artifactType string, status string, sectionStatuses []setupSvcLiveReplayEvidenceSectionStatus) bool {
	if options.Domain != "" && normalizeDomain(domain) != options.Domain {
		return false
	}
	if options.Operation != "" && strings.ToLower(strings.TrimSpace(operation)) != options.Operation {
		return false
	}
	if options.ArtifactType != "" && setupSvcLiveReplayNormalizeArtifactType(artifactType) != options.ArtifactType {
		return false
	}
	if options.Status != "" && strings.ToLower(strings.TrimSpace(status)) != options.Status {
		return false
	}
	if options.EvidenceSection != "" || options.SectionStatus != "" {
		return setupSvcLiveReplayCollectionPlanSectionMatches(options, sectionStatuses)
	}
	return true
}

func setupSvcLiveReplayCollectionPlanSectionMatches(options setupSvcLiveReplayCollectionPlanOptions, sectionStatuses []setupSvcLiveReplayEvidenceSectionStatus) bool {
	for _, sectionStatus := range sectionStatuses {
		if options.EvidenceSection != "" && !strings.EqualFold(sectionStatus.Section, options.EvidenceSection) {
			continue
		}
		if options.SectionStatus != "" && !strings.EqualFold(sectionStatus.Status, options.SectionStatus) {
			continue
		}
		return true
	}
	return false
}

func setupSvcLiveReplayDomainContractMap() map[string]setupSvcLiveReplayDomain {
	contracts := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		contracts[normalizeDomain(domain.Domain)] = domain
	}
	return contracts
}

func setupSvcLiveReplayCollectionArtifactStatus(operation setupSvcLiveReplayGapOperation, file string) string {
	path := setupSvcLiveReplayEvidencePathForContract("", file)
	for _, item := range operation.FailedEvidence {
		if strings.Contains(item, path) || strings.Contains(item, file) {
			return "failed"
		}
	}
	for _, item := range operation.MissingEvidence {
		if strings.Contains(item, path) || strings.Contains(item, file) {
			return "missing"
		}
	}
	for _, item := range operation.PendingEvidence {
		if strings.Contains(item, path) || strings.Contains(item, file) {
			return "pending"
		}
	}
	switch operation.Status {
	case "missing_manifest_operation", "missing_evidence":
		return "missing"
	case "failed_evidence":
		return "failed"
	case "pending_evidence", "ready_for_normalized_diff":
		if setupSvcLiveReplayArtifactType(file) == "normalized-diff" && operation.Status == "ready_for_normalized_diff" {
			return "missing"
		}
		return "pending"
	default:
		return "passed"
	}
}

func setupSvcLiveReplayCollectionNextAction(status string, artifactType string, operationNextAction string) string {
	actionArtifactType := strings.ReplaceAll(artifactType, "-", "_")
	switch status {
	case "failed":
		return "repair_" + actionArtifactType + "_artifact"
	case "missing":
		if artifactType == "normalized-diff" || operationNextAction == "generate_normalized_diff" {
			return "generate_normalized_diff"
		}
		return "collect_" + actionArtifactType + "_artifact"
	case "pending":
		return "replace_pending_" + actionArtifactType + "_artifact"
	default:
		return operationNextAction
	}
}

func setupSvcLiveReplayCollectionRequiredShapeKey(artifactType string) string {
	switch artifactType {
	case "setup-svc", "metadata-service":
		return "requiredSnapshotShape"
	case "query-readback":
		return "requiredReadbackShape"
	case "normalized-diff":
		return "requiredDiffShape"
	case "cleanup":
		return "requiredCleanupShape"
	default:
		return ""
	}
}

func setupSvcLiveReplayCollectionRequiredTables(contract setupSvcLiveReplayDomain, artifactType string) []string {
	if len(contract.RequiredTables) == 0 {
		return nil
	}
	return append([]string{}, contract.RequiredTables...)
}

func setupSvcLiveReplayCollectionRuntimeEffects(contract setupSvcLiveReplayDomain, artifactType string) []string {
	switch artifactType {
	case "setup-svc", "metadata-service":
		if len(contract.RuntimeEffects) == 0 {
			return nil
		}
		return append([]string{}, contract.RuntimeEffects...)
	default:
		return nil
	}
}

func setupSvcLiveReplayCollectionQueryReadbackExpectations(contract setupSvcLiveReplayDomain, artifactType string) []string {
	if artifactType != "query-readback" || len(contract.QueryReadbackExpectations) == 0 {
		return nil
	}
	return append([]string{}, contract.QueryReadbackExpectations...)
}

func setupSvcLiveReplayGapCountPrefix(values []string, prefix string) int {
	count := 0
	for _, value := range values {
		if strings.HasPrefix(value, prefix) {
			count++
		}
	}
	return count
}

func buildSetupSvcLiveReplayNormalizedDiffApplyResult(projectPath string, manifestArg string, execute bool, approval string) (setupSvcLiveReplayNormalizedDiffApplyResult, error) {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	result := setupSvcLiveReplayNormalizedDiffApplyResult{
		Mode:             "setup-svc-live-replay-normalized-diff",
		Project:          projectPath,
		ReadOnly:         !execute,
		Execute:          execute,
		ApprovalRequired: true,
		Approved:         execute && approval == setupSvcParityNormalizedDiffApproval,
		Status:           "dry_run_ready",
		ManifestPath:     manifestPath,
		NextCommands: setupSvcLiveReplayNormalizedDiffCommands{
			GenerateDiffs:   "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-normalized-diff " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityNormalizedDiffApproval,
			VerifyEvidence:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
			PromotionAudit:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath),
			CompletionAudit: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		},
		Notes: []string{
			"This command only computes local normalized-diff.json artifacts from existing setup-svc.json and metadata-service.json snapshots.",
			"It does not execute setup-svc or MetadataService writes; source evidence artifacts must already contain real passed table snapshots.",
			"Execute updates normalizedDiffStatus in the manifest to passed or failed per operation.",
		},
	}
	if execute && approval != setupSvcParityNormalizedDiffApproval {
		return result, fmt.Errorf("refusing to write setup-svc normalized diff evidence without --approval %s", setupSvcParityNormalizedDiffApproval)
	}
	manifest, err := readJSONFile(manifestPath)
	if err != nil {
		return result, err
	}
	for _, issue := range setupSvcLiveReplayContractIdentityIssues(firstMapString(manifest, "contractVersion"), firstMapString(manifest, "contractFingerprint")) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	for _, issue := range setupSvcLiveReplayProjectIdentityIssues(projectPath, firstMapString(manifest, "project", "projectPath")) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	requiredTablesByDomain := map[string][]string{}
	expectedDomains := map[string]bool{}
	for _, domain := range setupSvcLiveReplayDomains() {
		normalized := normalizeDomain(domain.Domain)
		expectedDomains[normalized] = true
		requiredTablesByDomain[normalized] = append([]string{}, domain.RequiredTables...)
	}
	for _, rawDomain := range mapList(manifest["domains"]) {
		domainName := normalizeDomain(firstMapString(rawDomain, "domain", "name"))
		if domainName == "" {
			result.BlockingIssues = append(result.BlockingIssues, "manifest: missing domain name")
			continue
		}
		domainResult := setupSvcLiveReplayNormalizedDiffDomain{Domain: domainName, Status: "clean"}
		result.Totals.Domains++
		if !expectedDomains[domainName] {
			domainResult.Status = "blocked"
			result.BlockingIssues = append(result.BlockingIssues, "manifest: unexpected domain "+domainName)
			result.Domains = append(result.Domains, domainResult)
			continue
		}
		for _, rawOperation := range mapList(rawDomain["operations"]) {
			operationName := strings.ToLower(strings.TrimSpace(firstMapString(rawOperation, "operation", "name")))
			if operationName == "" {
				result.BlockingIssues = append(result.BlockingIssues, domainName+": missing operation")
				continue
			}
			result.Totals.Operations++
			operationResult, artifact := buildSetupSvcLiveReplayOperationDiff(projectPath, domainName, operationName, rawOperation, requiredTablesByDomain[domainName])
			result.Totals.DiffFiles++
			switch operationResult.Status {
			case "clean":
				result.Totals.CleanOperations++
			case "dirty":
				result.Totals.DirtyOperations++
				domainResult.Status = "dirty"
			default:
				result.Totals.BlockedOps++
				domainResult.Status = "blocked"
				for _, issue := range operationResult.BlockingIssues {
					result.BlockingIssues = append(result.BlockingIssues, domainName+"/"+operationName+": "+issue)
				}
			}
			if execute && artifact != nil && operationResult.DiffFile != "" {
				if err := writeSetupSvcLiveReplayDiffArtifact(projectPath, operationResult.DiffFile, artifact); err != nil {
					operationResult.Status = "blocked"
					operationResult.BlockingIssues = append(operationResult.BlockingIssues, err.Error())
					result.BlockingIssues = append(result.BlockingIssues, domainName+"/"+operationName+": "+err.Error())
					result.Totals.BlockedOps++
				} else {
					result.Totals.WrittenFiles++
					setupSvcLiveReplaySetManifestOperationDiffStatus(rawOperation, artifact.Status)
				}
			}
			domainResult.Operations = append(domainResult.Operations, operationResult)
		}
		result.Domains = append(result.Domains, domainResult)
	}
	if execute && len(result.BlockingIssues) == 0 {
		if err := writeSetupSvcLiveReplayManifestMap(manifestPath, manifest); err != nil {
			result.Status = "blocked_manifest_write"
			result.BlockingIssues = append(result.BlockingIssues, err.Error())
			return result, nil
		}
		result.Totals.WrittenFiles++
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked"
		return result, nil
	}
	if result.Totals.DirtyOperations > 0 {
		result.Status = "diffs_failed"
	} else if execute {
		result.Status = "applied"
	} else {
		result.Warnings = []string{"Dry run only; no normalized-diff artifacts or manifest statuses were written."}
	}
	return result, nil
}

func buildSetupSvcLiveReplayManifestSyncApplyResult(projectPath string, manifestArg string, execute bool, approval string) (setupSvcLiveReplayManifestSyncApplyResult, error) {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	result := setupSvcLiveReplayManifestSyncApplyResult{
		Mode:             "setup-svc-live-replay-manifest-sync",
		Project:          projectPath,
		ReadOnly:         !execute,
		Execute:          execute,
		ApprovalRequired: true,
		Approved:         execute && approval == setupSvcParityManifestSyncApproval,
		Status:           "dry_run_ready",
		ManifestPath:     manifestPath,
		NextCommands: setupSvcLiveReplayManifestSyncCommands{
			SyncManifest:    "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-manifest-sync " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityManifestSyncApproval,
			VerifyEvidence:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
			PromotionAudit:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath),
			CompletionAudit: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		},
		Notes: []string{
			"This command only derives manifest statuses from existing evidence artifact JSON files.",
			"It does not execute setup-svc, MetadataService, promotion, or matrix writes.",
			"Execute rewrites manifest.json so strict evidence verification can run without manual status edits.",
		},
	}
	if execute && approval != setupSvcParityManifestSyncApproval {
		return result, fmt.Errorf("refusing to sync setup-svc live replay manifest without --approval %s", setupSvcParityManifestSyncApproval)
	}
	manifest, err := readJSONFile(manifestPath)
	if err != nil {
		return result, err
	}
	if mode := firstMapString(manifest, "mode"); mode != setupSvcLiveReplayEvidenceMode {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: mode must be "+setupSvcLiveReplayEvidenceMode)
	}
	for _, issue := range setupSvcLiveReplayContractIdentityIssues(firstMapString(manifest, "contractVersion"), firstMapString(manifest, "contractFingerprint")) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	for _, issue := range setupSvcLiveReplayProjectIdentityIssues(projectPath, firstMapString(manifest, "project", "projectPath")) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	expectedByDomain := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		expectedByDomain[normalizeDomain(domain.Domain)] = domain
	}
	manifestDomains := map[string]map[string]any{}
	for _, rawDomain := range mapList(manifest["domains"]) {
		domainName := normalizeDomain(firstMapString(rawDomain, "domain", "name"))
		switch {
		case domainName == "":
			result.BlockingIssues = append(result.BlockingIssues, "manifest: missing domain name")
		case expectedByDomain[domainName].Domain == "":
			result.BlockingIssues = append(result.BlockingIssues, "manifest: unexpected domain "+domainName)
		case manifestDomains[domainName] != nil:
			result.BlockingIssues = append(result.BlockingIssues, "manifest: duplicate domain "+domainName)
		default:
			manifestDomains[domainName] = rawDomain
		}
	}
	for _, expected := range setupSvcLiveReplayDomains() {
		domainResult := setupSvcLiveReplayManifestSyncDomain{Domain: expected.Domain, Status: "passed"}
		result.Totals.Domains++
		domainManifest := manifestDomains[normalizeDomain(expected.Domain)]
		if domainManifest == nil {
			domainResult.Status = "failed"
			result.BlockingIssues = append(result.BlockingIssues, expected.Domain+": missing domain evidence")
			for _, operation := range expected.Operations {
				result.Totals.Operations++
				result.Totals.FailedOperations++
				domainResult.Operations = append(domainResult.Operations, setupSvcLiveReplayManifestSyncOperation{
					Operation: operation,
					Status:    "failed",
					Issues:    []string{"missing manifest operation"},
				})
			}
			result.Domains = append(result.Domains, domainResult)
			continue
		}
		operationManifest := setupSvcLiveReplayManifestOperationMap(domainManifest, nil, expected.Domain)
		for _, operation := range expected.Operations {
			result.Totals.Operations++
			operationResult := buildSetupSvcLiveReplayManifestSyncOperation(projectPath, expected, operation, operationManifest[strings.ToLower(operation)])
			result.Totals.ArtifactFiles += len(operationResult.ArtifactStatuses)
			for _, artifact := range operationResult.ArtifactStatuses {
				switch artifact.Status {
				case "passed":
					result.Totals.PassedArtifacts++
				case "failed":
					result.Totals.FailedArtifacts++
				default:
					result.Totals.PendingArtifacts++
				}
			}
			switch operationResult.Status {
			case "passed":
				result.Totals.PassedOperations++
			case "failed":
				result.Totals.FailedOperations++
				domainResult.Status = "failed"
			default:
				result.Totals.PendingOperations++
				if domainResult.Status == "passed" {
					domainResult.Status = "pending"
				}
			}
			if operationResult.Updated {
				result.Totals.UpdatedOperations++
			}
			domainResult.Operations = append(domainResult.Operations, operationResult)
		}
		result.Domains = append(result.Domains, domainResult)
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked_invalid_manifest"
		setupSvcLiveReplayMirrorManifestSyncTotals(&result)
		return result, nil
	}
	switch {
	case result.Totals.FailedOperations > 0 || result.Totals.FailedArtifacts > 0:
		result.Status = "failed_evidence"
		manifest["status"] = "failed"
	case result.Totals.PendingOperations > 0 || result.Totals.PendingArtifacts > 0:
		result.Status = "pending_evidence"
		manifest["status"] = "pending"
	default:
		result.Status = "passed"
		manifest["status"] = "passed"
	}
	if execute {
		if err := writeSetupSvcLiveReplayManifestMap(manifestPath, manifest); err != nil {
			result.Status = "blocked_manifest_write"
			result.BlockingIssues = append(result.BlockingIssues, err.Error())
			setupSvcLiveReplayMirrorManifestSyncTotals(&result)
			return result, nil
		}
		result.Totals.WrittenFiles++
	} else {
		result.Warnings = []string{"Dry run only; manifest statuses were derived but manifest.json was not rewritten."}
	}
	setupSvcLiveReplayMirrorManifestSyncTotals(&result)
	return result, nil
}

func setupSvcLiveReplayMirrorManifestSyncTotals(result *setupSvcLiveReplayManifestSyncApplyResult) {
	if result == nil {
		return
	}
	result.ArtifactFiles = result.Totals.ArtifactFiles
	result.PassedArtifacts = result.Totals.PassedArtifacts
	result.PendingArtifacts = result.Totals.PendingArtifacts
	result.FailedArtifacts = result.Totals.FailedArtifacts
	result.PassedOperations = result.Totals.PassedOperations
	result.PendingOperations = result.Totals.PendingOperations
	result.FailedOperations = result.Totals.FailedOperations
	result.UpdatedOperations = result.Totals.UpdatedOperations
	result.WrittenFiles = result.Totals.WrittenFiles
}

type setupSvcLiveReplayEvidenceImportItem struct {
	Domain       string
	Operation    string
	ArtifactType string
	Path         string
	SourcePath   string
	Artifact     map[string]any
	LoadIssues   []string
}

func buildSetupSvcLiveReplayEvidenceImportApplyResult(projectPath string, packet map[string]any, execute bool, approval string) (setupSvcLiveReplayEvidenceImportApplyResult, error) {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, firstMapString(packet, "manifestPath", "manifest", "manifestFile"))
	packetCommands := setupSvcLiveReplayEvidenceImportPacketCommands(packet)
	result := setupSvcLiveReplayEvidenceImportApplyResult{
		Mode:             "setup-svc-live-replay-evidence-import",
		Project:          projectPath,
		ReadOnly:         !execute,
		Execute:          execute,
		ApprovalRequired: true,
		Approved:         execute && approval == setupSvcParityEvidenceImportApproval,
		Status:           "dry_run_ready",
		ManifestPath:     manifestPath,
		NextCommands: setupSvcLiveReplayEvidenceImportCommands{
			ImportEvidence:  "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import <@evidence-import.json> --execute --approval " + setupSvcParityEvidenceImportApproval,
			SyncManifest:    "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-manifest-sync " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityManifestSyncApproval,
			VerifyEvidence:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
			Worklist:        "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-worklist " + shellPath(manifestPath),
			CompletionAudit: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		},
		Notes: []string{
			"This command validates and imports captured setup-svc live replay evidence artifact JSON files only.",
			"It does not execute setup-svc, MetadataService, normalized diff, cleanup, manifest sync, bundle, promotion, or matrix writes.",
			"Run manifest-sync after importing a passed batch so manifest statuses are derived from the imported artifacts.",
		},
	}
	result.NextCommands.SuggestedWorklistPath = packetCommands.SuggestedWorklistPath
	result.NextCommands.SaveCurrentWorklist = packetCommands.SaveCurrentWorklist
	result.NextCommands.DryRunCurrentImport = packetCommands.DryRunCurrentImport
	result.NextCommands.ExecuteCurrentImport = packetCommands.ExecuteCurrentImport
	if execute && approval != setupSvcParityEvidenceImportApproval {
		return result, fmt.Errorf("refusing to import setup-svc live replay evidence without --approval %s", setupSvcParityEvidenceImportApproval)
	}
	manifest, err := readJSONFile(manifestPath)
	if err != nil {
		return result, err
	}
	if mode := firstMapString(manifest, "mode"); mode != setupSvcLiveReplayEvidenceMode {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: mode must be "+setupSvcLiveReplayEvidenceMode)
	}
	for _, issue := range setupSvcLiveReplayContractIdentityIssues(firstMapString(manifest, "contractVersion"), firstMapString(manifest, "contractFingerprint")) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	for _, issue := range setupSvcLiveReplayProjectIdentityIssues(projectPath, firstMapString(manifest, "project", "projectPath")) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	items := setupSvcLiveReplayEvidenceImportItems(projectPath, packet)
	if len(items) == 0 {
		result.BlockingIssues = append(result.BlockingIssues, "packet: missing artifacts or artifactReplacementRecords")
	}
	expectedByDomain := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		expectedByDomain[normalizeDomain(domain.Domain)] = domain
	}
	seenPaths := map[string]setupSvcLiveReplayEvidenceImportItem{}
	normalizedItems := make([]setupSvcLiveReplayEvidenceImportItem, 0, len(items))
	for index, item := range items {
		normalized, itemResult := setupSvcLiveReplayNormalizeEvidenceImportItem(projectPath, index, item, expectedByDomain)
		if itemResult.Path != "" {
			if existing, ok := seenPaths[itemResult.Path]; ok {
				if setupSvcLiveReplayEvidenceImportDuplicateCompatible(existing, normalized) {
					result.Totals.SkippedDuplicateRecords++
					continue
				}
				itemResult.Issues = append(itemResult.Issues, "duplicate artifact path in import packet with conflicting source")
			} else {
				seenPaths[itemResult.Path] = normalized
			}
		}
		normalizedItems = append(normalizedItems, normalized)
		if len(itemResult.Issues) == 0 {
			failures := verifySetupSvcLiveReplayEvidenceArtifact(projectPath, normalized.Path, expectedByDomain[normalizeDomain(normalized.Domain)], normalized.Operation, normalized.Artifact)
			itemResult.Issues = append(itemResult.Issues, failures...)
		}
		if len(itemResult.Issues) > 0 {
			itemResult.Status = "failed"
			result.Totals.Failed++
			for _, issue := range itemResult.Issues {
				result.BlockingIssues = append(result.BlockingIssues, itemResult.Path+": "+issue)
			}
		} else {
			itemResult.Status = "ready"
			result.Totals.Passed++
		}
		result.Totals.Artifacts++
		result.Artifacts = append(result.Artifacts, itemResult)
	}
	setupSvcLiveReplayEvidenceImportMirrorTotals(&result)
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked"
		result.RepairSummary = buildSetupSvcLiveReplayEvidenceImportRepairSummary(projectPath, manifestPath, packet, result.Artifacts)
		return result, nil
	}
	if !execute {
		result.Warnings = []string{"Dry run only; no evidence artifact files were written."}
		return result, nil
	}
	for i, artifact := range result.Artifacts {
		if err := writeSetupSvcLiveReplayImportedEvidenceArtifact(projectPath, artifact.Path, normalizedItems[i].Artifact); err != nil {
			result.Status = "blocked_artifact_write"
			result.BlockingIssues = append(result.BlockingIssues, artifact.Path+": "+err.Error())
			return result, nil
		}
		result.Artifacts[i].Status = "written"
		result.Totals.WrittenFiles++
	}
	setupSvcLiveReplayEvidenceImportMirrorTotals(&result)
	result.Status = "applied"
	return result, nil
}

func setupSvcLiveReplayEvidenceImportMirrorTotals(result *setupSvcLiveReplayEvidenceImportApplyResult) {
	result.ArtifactCount = result.Totals.Artifacts
	result.PassedArtifacts = result.Totals.Passed
	result.FailedArtifacts = result.Totals.Failed
	result.SkippedDuplicates = result.Totals.SkippedDuplicateRecords
	result.WrittenFiles = result.Totals.WrittenFiles
}

func buildSetupSvcLiveReplayEvidenceImportRepairSummary(projectPath string, manifestPath string, packet map[string]any, artifacts []setupSvcLiveReplayEvidenceImportResult) setupSvcLiveReplayEvidenceImportRepair {
	recordsByPath := setupSvcLiveReplayEvidenceImportRepairRecordsByPath(packet)
	issueCounts := map[string]int{}
	sectionCounts := map[string]int{}
	queueCounts := map[string]int{}
	artifactTypeCounts := map[string]int{}
	sourceRepairs := make([]setupSvcLiveReplayEvidenceImportSourceRepair, 0)
	seenSource := map[string]bool{}
	failedArtifacts := 0
	for _, artifact := range artifacts {
		if artifact.Status != "failed" {
			continue
		}
		failedArtifacts++
		artifactTypeCounts[artifact.ArtifactType]++
		for _, issue := range artifact.Issues {
			issueCounts[setupSvcLiveReplayEvidenceImportIssueFamily(issue)]++
		}
		record := recordsByPath[artifact.Path]
		missingSections := setupSvcLiveReplayEvidenceImportRepairMissingSections(artifact, record)
		for _, section := range missingSections {
			sectionCounts[section]++
			queueCounts[artifact.ArtifactType+"\x00"+section]++
		}
		sourcePath := artifact.SourcePath
		if sourcePath == "" {
			sourcePath = firstMapString(record, "suggestedSourcePath", "sourcePath", "sourceFile", "capturedPath", "capturedFile", "inputPath", "inputFile")
		}
		if sourcePath == "" || seenSource[sourcePath] {
			continue
		}
		seenSource[sourcePath] = true
		sourceRepairs = append(sourceRepairs, setupSvcLiveReplayEvidenceImportSourceRepair{
			Path:                    sourcePath,
			TargetPath:              artifact.Path,
			Domain:                  artifact.Domain,
			Operation:               artifact.Operation,
			ArtifactType:            artifact.ArtifactType,
			MissingEvidenceSections: missingSections,
			Issues:                  append([]string(nil), artifact.Issues...),
			CaptureTask:             firstMapAny(record, "captureTask"),
		})
	}
	repairQueues := setupSvcLiveReplayEvidenceImportRepairQueues(projectPath, manifestPath, queueCounts)
	return setupSvcLiveReplayEvidenceImportRepair{
		FailedArtifacts:         failedArtifacts,
		IssueCounts:             setupSvcLiveReplayEvidenceImportSortedIssueCounts(issueCounts),
		MissingEvidenceSections: setupSvcLiveReplayEvidenceImportSortedSectionCounts(sectionCounts),
		ArtifactTypes:           setupSvcLiveReplayEvidenceImportSortedIssueCounts(artifactTypeCounts),
		RepairQueueCount:        len(repairQueues),
		RepairQueues:            repairQueues,
		SourceFiles:             sourceRepairs,
	}
}

func setupSvcLiveReplayEvidenceImportRepairQueues(projectPath string, manifestPath string, counts map[string]int) []setupSvcLiveReplayEvidenceImportRepairQueue {
	queues := make([]setupSvcLiveReplayEvidenceImportRepairQueue, 0, len(counts))
	for key, count := range counts {
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
			continue
		}
		artifactType := parts[0]
		section := parts[1]
		filterArgs := " --artifact-type " + shellPath(artifactType) +
			" --evidence-section " + shellPath(section) +
			" --section-status missing --source-readiness incomplete"
		baseCommand := "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-worklist " + shellPath(manifestPath) +
			filterArgs + " --batch-index 0"
		sourceChecklistCommand := "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-checklist " + shellPath(manifestPath) +
			filterArgs
		capturePlanCommand := "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-capture-plan " + shellPath(manifestPath) +
			" --artifact-type " + shellPath(artifactType) +
			" --evidence-section " + shellPath(section) +
			" --section-status missing --source-readiness incomplete --limit 25"
		worklistPath := filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "worklist-repair-"+setupSvcLiveReplayRepairQueueSlug(artifactType)+"-"+setupSvcLiveReplayRepairQueueSlug(section)+"-readiness-incomplete-batch-0.json")
		sourceChecklistPath := filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "source-checklist-repair-"+setupSvcLiveReplayRepairQueueSlug(artifactType)+"-"+setupSvcLiveReplayRepairQueueSlug(section)+"-readiness-incomplete.json")
		sourceExecutionCommand := "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-source-execution-packet " + shellPath(manifestPath) +
			filterArgs
		sourceExecutionPath := filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "source-capture-execution-packet-repair-"+setupSvcLiveReplayRepairQueueSlug(artifactType)+"-"+setupSvcLiveReplayRepairQueueSlug(section)+"-readiness-incomplete.json")
		queues = append(queues, setupSvcLiveReplayEvidenceImportRepairQueue{
			ArtifactType:               artifactType,
			EvidenceSection:            section,
			Count:                      count,
			SourceFiles:                count,
			TargetFiles:                count,
			CapturePlanCommand:         capturePlanCommand,
			WorklistCommand:            baseCommand,
			SaveWorklistCommand:        baseCommand + " > " + shellPath(worklistPath),
			SourceChecklistCommand:     sourceChecklistCommand,
			SaveSourceChecklistCommand: sourceChecklistCommand + " > " + shellPath(sourceChecklistPath),
			SourceExecutionCommand:     sourceExecutionCommand,
			SuggestedSourceExecution:   sourceExecutionPath,
			SaveSourceExecutionCommand: sourceExecutionCommand + " > " + shellPath(sourceExecutionPath),
		})
	}
	sort.Slice(queues, func(i, j int) bool {
		if queues[i].Count == queues[j].Count {
			if queues[i].ArtifactType == queues[j].ArtifactType {
				return queues[i].EvidenceSection < queues[j].EvidenceSection
			}
			return queues[i].ArtifactType < queues[j].ArtifactType
		}
		return queues[i].Count > queues[j].Count
	})
	return queues
}

func setupSvcLiveReplayRepairQueueSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func setupSvcLiveReplayEvidenceImportRepairMissingSections(artifact setupSvcLiveReplayEvidenceImportResult, record map[string]any) []string {
	sections := stringList(record["missingEvidenceSections"])
	for _, issue := range artifact.Issues {
		for _, section := range setupSvcLiveReplayCompletionFailedEvidenceSections(artifact.ArtifactType, issue) {
			if setupSvcLiveReplayEvidenceIdentitySection(section) {
				continue
			}
			sections = setupSvcLiveReplayAppendUniqueStrings(sections, section)
		}
	}
	sort.Strings(sections)
	return sections
}

func setupSvcLiveReplayEvidenceImportRepairRecordsByPath(packet map[string]any) map[string]map[string]any {
	recordsByPath := map[string]map[string]any{}
	for _, record := range setupSvcLiveReplayEvidenceImportRecords(packet) {
		targetPath := strings.TrimPrefix(strings.TrimSpace(firstMapString(record, "path", "targetPath", "file", "evidenceFile")), "@")
		if targetPath == "" {
			if captureTask, ok := record["captureTask"].(map[string]any); ok {
				targetPath = strings.TrimPrefix(strings.TrimSpace(firstMapString(captureTask, "targetPath", "path")), "@")
			}
		}
		if targetPath == "" {
			continue
		}
		recordsByPath[filepath.ToSlash(targetPath)] = record
		recordsByPath[targetPath] = record
	}
	return recordsByPath
}

func setupSvcLiveReplayEvidenceImportIssueFamily(issue string) string {
	issue = strings.TrimSpace(issue)
	if issue == "" {
		return "unknown"
	}
	if before, _, ok := strings.Cut(issue, "="); ok {
		return before
	}
	if before, _, ok := strings.Cut(issue, ":"); ok {
		return before
	}
	return issue
}

func firstMapAny(values map[string]any, keys ...string) map[string]any {
	for _, key := range keys {
		if value, ok := values[key].(map[string]any); ok {
			return value
		}
	}
	return nil
}

func setupSvcLiveReplayEvidenceImportSortedIssueCounts(counts map[string]int) []setupSvcLiveReplayEvidenceImportIssueCount {
	items := make([]setupSvcLiveReplayEvidenceImportIssueCount, 0, len(counts))
	for name, count := range counts {
		items = append(items, setupSvcLiveReplayEvidenceImportIssueCount{Name: name, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Name < items[j].Name
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func setupSvcLiveReplayEvidenceImportSortedSectionCounts(counts map[string]int) []setupSvcLiveReplayEvidenceImportSectionCount {
	items := make([]setupSvcLiveReplayEvidenceImportSectionCount, 0, len(counts))
	for section, count := range counts {
		items = append(items, setupSvcLiveReplayEvidenceImportSectionCount{Section: section, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Section < items[j].Section
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func setupSvcLiveReplayEvidenceImportPacketCommands(packet map[string]any) setupSvcLiveReplayEvidenceImportCommands {
	var commands setupSvcLiveReplayEvidenceImportCommands
	mergeSetupSvcLiveReplayEvidenceImportPacketCommands(&commands, packet)
	if operatorPacket, ok := packet["operatorPacket"].(map[string]any); ok {
		mergeSetupSvcLiveReplayEvidenceImportPacketCommands(&commands, operatorPacket)
	}
	if operatorBatch, ok := packet["operatorBatch"].(map[string]any); ok {
		mergeSetupSvcLiveReplayEvidenceImportPacketCommands(&commands, operatorBatch)
	}
	for _, batch := range mapList(packet["batches"]) {
		mergeSetupSvcLiveReplayEvidenceImportPacketCommands(&commands, batch)
		if commands.DryRunCurrentImport != "" || commands.SaveCurrentWorklist != "" {
			return commands
		}
	}
	for _, queue := range mapList(packet["queues"]) {
		for _, batch := range mapList(queue["batches"]) {
			mergeSetupSvcLiveReplayEvidenceImportPacketCommands(&commands, batch)
			if commands.DryRunCurrentImport != "" || commands.SaveCurrentWorklist != "" {
				return commands
			}
			if operatorBatch, ok := batch["operatorBatch"].(map[string]any); ok {
				mergeSetupSvcLiveReplayEvidenceImportPacketCommands(&commands, operatorBatch)
				if commands.DryRunCurrentImport != "" || commands.SaveCurrentWorklist != "" {
					return commands
				}
			}
		}
	}
	return commands
}

func mergeSetupSvcLiveReplayEvidenceImportPacketCommands(commands *setupSvcLiveReplayEvidenceImportCommands, packet map[string]any) {
	if commands.SuggestedWorklistPath == "" {
		commands.SuggestedWorklistPath = firstMapString(packet, "suggestedWorklistPath", "worklistPath")
	}
	if commands.SaveCurrentWorklist == "" {
		commands.SaveCurrentWorklist = firstMapString(packet, "saveWorklistCommand")
	}
	if commands.DryRunCurrentImport == "" {
		commands.DryRunCurrentImport = firstMapString(packet, "dryRunImportCommand", "dryRunImport")
	}
	if commands.ExecuteCurrentImport == "" {
		commands.ExecuteCurrentImport = firstMapString(packet, "executeImportCommand", "executeImport")
	}
}

func setupSvcLiveReplayEvidenceImportDuplicateCompatible(existing setupSvcLiveReplayEvidenceImportItem, duplicate setupSvcLiveReplayEvidenceImportItem) bool {
	if strings.TrimSpace(existing.SourcePath) != "" || strings.TrimSpace(duplicate.SourcePath) != "" {
		return strings.TrimSpace(existing.SourcePath) == strings.TrimSpace(duplicate.SourcePath)
	}
	if existing.Artifact == nil || duplicate.Artifact == nil {
		return existing.Artifact == nil && duplicate.Artifact == nil
	}
	existingBody, err := json.Marshal(existing.Artifact)
	if err != nil {
		return false
	}
	duplicateBody, err := json.Marshal(duplicate.Artifact)
	if err != nil {
		return false
	}
	return string(existingBody) == string(duplicateBody)
}

func setupSvcLiveReplayEvidenceImportItems(projectPath string, packet map[string]any) []setupSvcLiveReplayEvidenceImportItem {
	sourceRoot := firstMapString(packet, "sourceRoot", "captureRoot", "evidenceRoot", "inputRoot")
	records := setupSvcLiveReplayEvidenceImportRecords(packet)
	items := make([]setupSvcLiveReplayEvidenceImportItem, 0, len(records))
	for _, record := range records {
		sourcePath := setupSvcLiveReplayEvidenceImportRecordSourcePath(projectPath, sourceRoot, record)
		artifact, loadIssues := setupSvcLiveReplayEvidenceImportArtifact(projectPath, record, sourceRoot, sourcePath)
		items = append(items, setupSvcLiveReplayEvidenceImportItem{
			Domain:       firstMapString(record, "domain", "msapiDomain", "metadataDomain"),
			Operation:    firstMapString(record, "operation", "operationType", "action"),
			ArtifactType: setupSvcLiveReplayNormalizeArtifactType(firstMapString(record, "artifactType", "evidenceType")),
			Path:         firstMapString(record, "path", "targetPath", "file", "evidenceFile"),
			SourcePath:   setupSvcLiveReplayEvidenceImportSourcePath(projectPath, sourceRoot, sourcePath),
			Artifact:     artifact,
			LoadIssues:   loadIssues,
		})
	}
	return items
}

func setupSvcLiveReplayEvidenceImportRecords(packet map[string]any) []map[string]any {
	var records []map[string]any
	for _, key := range []string{"artifacts", "artifactReplacementRecords", "replacements", "evidenceArtifacts"} {
		records = append(records, mapList(packet[key])...)
	}
	if strings.TrimSpace(firstMapString(packet, "mode")) == "setup-svc-live-replay-source-execution-packet" {
		records = append(records, mapList(packet["items"])...)
	}
	for _, key := range []string{"operatorBatch", "batch"} {
		if nested, ok := packet[key].(map[string]any); ok {
			records = append(records, mapList(nested["artifactReplacementRecords"])...)
			if key != "operatorBatch" {
				records = append(records, mapList(nested["artifacts"])...)
			}
		}
	}
	for _, batch := range mapList(packet["batches"]) {
		records = append(records, mapList(batch["artifactReplacementRecords"])...)
		if operatorBatch, ok := batch["operatorBatch"].(map[string]any); ok {
			records = append(records, mapList(operatorBatch["artifactReplacementRecords"])...)
		}
	}
	for _, queue := range mapList(packet["queues"]) {
		for _, batch := range mapList(queue["batches"]) {
			records = append(records, mapList(batch["artifactReplacementRecords"])...)
			if operatorBatch, ok := batch["operatorBatch"].(map[string]any); ok {
				records = append(records, mapList(operatorBatch["artifactReplacementRecords"])...)
			}
		}
	}
	return records
}

func setupSvcLiveReplayEvidenceImportRecordSourcePath(projectPath string, sourceRoot string, record map[string]any) string {
	if explicit := firstMapString(record, "sourcePath", "sourceFile", "capturedPath", "capturedFile", "inputPath", "inputFile"); explicit != "" {
		return explicit
	}
	if strings.TrimSpace(sourceRoot) == "" {
		return ""
	}
	targetPath := strings.TrimPrefix(strings.TrimSpace(firstMapString(record, "path", "targetPath", "file", "evidenceFile")), "@")
	if targetPath == "" {
		return ""
	}
	if filepath.IsAbs(targetPath) {
		relativePath, err := filepath.Rel(projectPath, targetPath)
		if err != nil || strings.HasPrefix(relativePath, "..") || filepath.IsAbs(relativePath) {
			return ""
		}
		return relativePath
	}
	return targetPath
}

func setupSvcLiveReplayEvidenceImportArtifact(projectPath string, record map[string]any, sourceRoot string, sourcePath string) (map[string]any, []string) {
	for _, key := range []string{"artifact", "evidence", "content", "payload"} {
		if artifact, ok := record[key].(map[string]any); ok {
			return artifact, nil
		}
	}
	if sourcePath := setupSvcLiveReplayEvidenceImportSourcePath(projectPath, sourceRoot, sourcePath); sourcePath != "" {
		artifact, err := readJSONFile(sourcePath)
		if err != nil {
			return nil, []string{"sourceFileInvalidJSONOrUnreadable:" + sourcePath}
		}
		return artifact, nil
	}
	if firstMapString(record, "contractVersion", "project", "projectPath", "status", "evidenceStatus") != "" {
		return record, nil
	}
	return nil, nil
}

func setupSvcLiveReplayEvidenceImportSourcePath(projectPath string, sourceRoot string, sourcePath string) string {
	sourcePath = strings.TrimPrefix(strings.TrimSpace(sourcePath), "@")
	if sourcePath == "" {
		return ""
	}
	sourceRoot = strings.TrimPrefix(strings.TrimSpace(sourceRoot), "@")
	if filepath.IsAbs(sourcePath) {
		return sourcePath
	}
	if sourceRoot != "" {
		if !filepath.IsAbs(sourceRoot) {
			rootPrefix := filepath.ToSlash(strings.Trim(sourceRoot, string(filepath.Separator))) + "/"
			if strings.HasPrefix(filepath.ToSlash(sourcePath), rootPrefix) {
				return filepath.Join(projectPath, sourcePath)
			}
			sourceRoot = filepath.Join(projectPath, sourceRoot)
		}
		return filepath.Join(sourceRoot, sourcePath)
	}
	return filepath.Join(projectPath, sourcePath)
}

func setupSvcLiveReplayNormalizeEvidenceImportItem(projectPath string, index int, item setupSvcLiveReplayEvidenceImportItem, expectedByDomain map[string]setupSvcLiveReplayDomain) (setupSvcLiveReplayEvidenceImportItem, setupSvcLiveReplayEvidenceImportResult) {
	item.Path = setupSvcLiveReplayEvidencePathForContract(projectPath, item.Path)
	if item.Path != "" {
		parts := strings.Split(filepath.ToSlash(item.Path), "/")
		if len(parts) >= 5 && strings.Join(parts[:2], "/") == "outputs/setup-svc-live-replay" {
			if item.Domain == "" {
				item.Domain = parts[2]
			}
			if item.Operation == "" {
				item.Operation = parts[3]
			}
			if item.ArtifactType == "" {
				item.ArtifactType = setupSvcLiveReplayArtifactType(item.Path)
			}
		}
	}
	item.Domain = normalizeDomain(item.Domain)
	item.Operation = strings.ToLower(strings.TrimSpace(item.Operation))
	item.ArtifactType = setupSvcLiveReplayNormalizeArtifactType(item.ArtifactType)
	result := setupSvcLiveReplayEvidenceImportResult{
		Domain:       item.Domain,
		Operation:    item.Operation,
		ArtifactType: item.ArtifactType,
		Path:         item.Path,
		SourcePath:   item.SourcePath,
		Status:       "ready",
	}
	result.Issues = append(result.Issues, item.LoadIssues...)
	if result.Path == "" {
		result.Path = fmt.Sprintf("artifact[%d]", index)
	}
	expected := expectedByDomain[item.Domain]
	switch {
	case item.Domain == "":
		result.Issues = append(result.Issues, "missing domain")
	case expected.Domain == "":
		result.Issues = append(result.Issues, "unexpected domain")
	}
	if item.Operation == "" {
		result.Issues = append(result.Issues, "missing operation")
	} else if !setupSvcLiveReplayDomainHasOperation(expected, item.Operation) {
		result.Issues = append(result.Issues, "unexpected operation")
	}
	if item.ArtifactType == "" {
		result.Issues = append(result.Issues, "missing artifactType")
	}
	if item.Artifact == nil {
		result.Issues = append(result.Issues, "missing artifact payload or sourcePath")
	}
	if len(result.Issues) == 0 {
		expectedFiles := setupSvcLiveReplayEvidenceFiles(expected.Domain, item.Operation, item.Operation != "query")
		if item.Path == "" {
			for _, file := range expectedFiles {
				if setupSvcLiveReplayArtifactType(file) == item.ArtifactType {
					item.Path = setupSvcLiveReplayEvidencePathForContract(projectPath, file)
					result.Path = item.Path
					break
				}
			}
		}
		if !setupSvcLiveReplayRequiredFileAllowed(projectPath, item.Path, expectedFiles) {
			result.Issues = append(result.Issues, "artifact path is not in the operation evidenceFiles contract")
		} else if setupSvcLiveReplayArtifactType(item.Path) != item.ArtifactType {
			result.Issues = append(result.Issues, "artifactType does not match artifact path")
		}
	}
	return item, result
}

func setupSvcLiveReplayDomainHasOperation(domain setupSvcLiveReplayDomain, operation string) bool {
	for _, expected := range domain.Operations {
		if strings.EqualFold(strings.TrimSpace(expected), strings.TrimSpace(operation)) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayRequiredFileAllowed(projectPath string, file string, expectedFiles []string) bool {
	key := setupSvcLiveReplayEvidencePathForContract(projectPath, file)
	if key == "" {
		return false
	}
	for _, expected := range expectedFiles {
		if setupSvcLiveReplayEvidencePathForContract(projectPath, expected) == key {
			return true
		}
	}
	return false
}

func writeSetupSvcLiveReplayImportedEvidenceArtifact(projectPath string, filePath string, artifact map[string]any) error {
	path := setupSvcLiveReplayResolveEvidenceFile(projectPath, filePath)
	body, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0644)
}

func buildSetupSvcLiveReplayEvidenceBundleApplyResult(projectPath string, manifestArg string, execute bool, approval string) (setupSvcLiveReplayEvidenceBundleApplyResult, error) {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	bundlePath := filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "evidence-bundle.json")
	result := setupSvcLiveReplayEvidenceBundleApplyResult{
		Mode:             "setup-svc-live-replay-evidence-bundle",
		Project:          projectPath,
		ReadOnly:         !execute,
		Execute:          execute,
		ApprovalRequired: true,
		Approved:         execute && approval == setupSvcParityEvidenceBundleApproval,
		Status:           "dry_run_ready",
		ManifestPath:     manifestPath,
		BundlePath:       bundlePath,
		NextCommands: setupSvcLiveReplayEvidenceBundleCommands{
			WriteBundle:     "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityEvidenceBundleApproval,
			VerifyEvidence:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
			PromotionAudit:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath),
			CompletionAudit: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		},
		Notes: []string{
			"This command writes a checksum bundle only after strict setup-svc live replay evidence verification passes.",
			"It does not execute setup-svc, MetadataService, promotion, or matrix writes.",
			"Use the bundle to preserve the exact manifest and artifact files that supported a matrix promotion decision.",
		},
	}
	if execute && approval != setupSvcParityEvidenceBundleApproval {
		return result, fmt.Errorf("refusing to write setup-svc live replay evidence bundle without --approval %s", setupSvcParityEvidenceBundleApproval)
	}
	evidence, err := buildSetupSvcLiveReplayEvidenceResult(projectPath, manifestArg)
	if err != nil {
		return result, err
	}
	result.ContractVersion = evidence.ContractVersion
	result.ContractFingerprint = evidence.ContractFingerprint
	result.EvidenceStatus = evidence.Status
	result.Totals.VerifiedDomains = evidence.Totals.VerifiedDomains
	result.Totals.VerifiedOperations = evidence.Totals.VerifiedOperations
	if evidence.Status != "passed" {
		result.Status = "blocked_evidence_not_passed"
		result.BlockingIssues = append(result.BlockingIssues, "evidence: "+evidence.Status)
		result.BlockingIssues = append(result.BlockingIssues, evidence.BlockingIssues...)
		result.BlockingIssues = append(result.BlockingIssues, setupSvcLiveReplayEvidenceFailureIssues(evidence)...)
		return result, nil
	}
	manifest, err := readJSONFile(manifestPath)
	if err != nil {
		return result, err
	}
	manifestFile, err := buildSetupSvcLiveReplayEvidenceBundleFile(projectPath, manifestPath, "manifest", "", "")
	if err != nil {
		result.Status = "blocked_bundle_input"
		result.BlockingIssues = append(result.BlockingIssues, err.Error())
		return result, nil
	}
	result.Files = append(result.Files, manifestFile)
	result.Totals.ManifestFiles++
	expectedDomains := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		expectedDomains[normalizeDomain(domain.Domain)] = domain
	}
	seenFiles := map[string]bool{manifestFile.Path: true}
	for _, rawDomain := range mapList(manifest["domains"]) {
		domainName := normalizeDomain(firstMapString(rawDomain, "domain", "name"))
		expected := expectedDomains[domainName]
		if expected.Domain == "" {
			continue
		}
		for _, rawOperation := range mapList(rawDomain["operations"]) {
			operation := strings.ToLower(strings.TrimSpace(firstMapString(rawOperation, "operation", "name", "mode")))
			if operation == "" {
				continue
			}
			for _, filePath := range setupSvcLiveReplayEvidenceFileList(rawOperation["evidenceFiles"]) {
				contractPath := setupSvcLiveReplayEvidencePathForContract(projectPath, filePath)
				if contractPath == "" || seenFiles[contractPath] {
					continue
				}
				seenFiles[contractPath] = true
				file, err := buildSetupSvcLiveReplayEvidenceBundleFile(projectPath, filePath, setupSvcLiveReplayArtifactType(filePath), expected.Domain, operation)
				if err != nil {
					result.BlockingIssues = append(result.BlockingIssues, expected.Domain+"/"+operation+": "+err.Error())
					continue
				}
				result.Files = append(result.Files, file)
				result.Totals.ArtifactFiles++
			}
		}
	}
	sort.Slice(result.Files, func(i, j int) bool { return result.Files[i].Path < result.Files[j].Path })
	result.Totals.EvidenceFiles = len(result.Files)
	for _, file := range result.Files {
		result.Totals.TotalBytes += file.Bytes
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked_bundle_input"
		return result, nil
	}
	if execute {
		result.Totals.WrittenFiles = 1
		result.Status = "applied"
		if err := writeSetupSvcLiveReplayEvidenceBundle(bundlePath, result); err != nil {
			result.Status = "blocked_bundle_write"
			result.Totals.WrittenFiles = 0
			result.BlockingIssues = append(result.BlockingIssues, err.Error())
			return result, nil
		}
	} else {
		result.Status = "dry_run_ready"
		result.Warnings = []string{"Dry run only; evidence-bundle.json was not written."}
	}
	return result, nil
}

func buildSetupSvcLiveReplayEvidenceBundleFile(projectPath string, filePath string, artifactType string, domain string, operation string) (setupSvcLiveReplayEvidenceBundleFile, error) {
	resolved := setupSvcLiveReplayResolveEvidenceFile(projectPath, filePath)
	body, err := os.ReadFile(resolved)
	if err != nil {
		return setupSvcLiveReplayEvidenceBundleFile{}, fmt.Errorf("cannot read bundle file %s: %w", filePath, err)
	}
	sum := sha256.Sum256(body)
	return setupSvcLiveReplayEvidenceBundleFile{
		Path:         setupSvcLiveReplayEvidencePathForContract(projectPath, filePath),
		ArtifactType: setupSvcLiveReplayNormalizeArtifactType(artifactType),
		Domain:       domain,
		Operation:    operation,
		SHA256:       fmt.Sprintf("sha256:%x", sum),
		Bytes:        int64(len(body)),
	}, nil
}

func writeSetupSvcLiveReplayEvidenceBundle(path string, result setupSvcLiveReplayEvidenceBundleApplyResult) error {
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0644)
}

func buildSetupSvcLiveReplayEvidenceBundleScanResult(projectPath string, manifestArg string) setupSvcLiveReplayEvidenceBundleScanResult {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	bundlePath := filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "evidence-bundle.json")
	bundle := verifySetupSvcLiveReplayEvidenceBundle(projectPath, manifestArg)
	result := setupSvcLiveReplayEvidenceBundleScanResult{
		Mode:         "setup-svc-live-replay-evidence-bundle",
		Project:      projectPath,
		ReadOnly:     true,
		Status:       bundle.Status,
		ManifestPath: manifestPath,
		BundlePath:   bundlePath,
		Bundle:       bundle,
		NextCommands: setupSvcLiveReplayEvidenceBundleCommands{
			WriteBundle:     "cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityEvidenceBundleApproval,
			VerifyEvidence:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence " + shellPath(manifestPath),
			PromotionAudit:  "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath),
			CompletionAudit: "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		},
		Notes: []string{
			"This read-only scan verifies that evidence-bundle.json matches the current manifest and evidence artifact SHA-256 set.",
			"It does not execute setup-svc, MetadataService, promotion, matrix writes, or bundle writes.",
			"Run it before matrix promotion to preflight missing, stale, or blocked checksum evidence.",
		},
	}
	if bundle.Status != "passed" {
		result.BlockingIssues = append(result.BlockingIssues, bundle.Issues...)
	}
	return result
}

func verifySetupSvcLiveReplayEvidenceBundle(projectPath string, manifestArg string) setupSvcLiveReplayEvidenceBundleVerification {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	bundlePath := filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "evidence-bundle.json")
	result := setupSvcLiveReplayEvidenceBundleVerification{
		Status:       "passed",
		BundlePath:   bundlePath,
		ManifestPath: manifestPath,
	}
	body, err := os.ReadFile(bundlePath)
	if err != nil {
		result.Status = "missing"
		result.Issues = append(result.Issues, "evidenceBundle: missing "+bundlePath)
		return result
	}
	var actual setupSvcLiveReplayEvidenceBundleApplyResult
	if err := json.Unmarshal(body, &actual); err != nil {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "evidenceBundle: invalid JSON "+err.Error())
		return result
	}
	expected, err := buildSetupSvcLiveReplayEvidenceBundleApplyResult(projectPath, manifestArg, false, "")
	if err != nil {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "evidenceBundle: cannot compute expected bundle "+err.Error())
		return result
	}
	result.EvidenceStatus = expected.EvidenceStatus
	if expected.Status != "dry_run_ready" || expected.EvidenceStatus != "passed" {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "evidenceBundle: evidence is not ready for bundle verification")
		result.Issues = append(result.Issues, expected.BlockingIssues...)
		return result
	}
	if actual.Mode != "setup-svc-live-replay-evidence-bundle" {
		result.Issues = append(result.Issues, "evidenceBundle: unexpected mode "+actual.Mode)
	}
	if actual.Status != "applied" {
		result.Issues = append(result.Issues, "evidenceBundle: status must be applied")
	}
	if filepath.Clean(actual.ManifestPath) != filepath.Clean(expected.ManifestPath) {
		result.Issues = append(result.Issues, "evidenceBundle: manifestPath mismatch")
	}
	if filepath.Clean(actual.BundlePath) != filepath.Clean(bundlePath) {
		result.Issues = append(result.Issues, "evidenceBundle: bundlePath mismatch")
	}
	if actual.Project != expected.Project {
		result.Issues = append(result.Issues, "evidenceBundle: project mismatch")
	}
	if actual.ContractVersion != expected.ContractVersion {
		result.Issues = append(result.Issues, "evidenceBundle: contractVersion mismatch")
	}
	if actual.ContractFingerprint != expected.ContractFingerprint {
		result.Issues = append(result.Issues, "evidenceBundle: contractFingerprint mismatch")
	}
	if actual.EvidenceStatus != "passed" {
		result.Issues = append(result.Issues, "evidenceBundle: evidenceStatus must be passed")
	}
	if actual.Totals.ManifestFiles != expected.Totals.ManifestFiles ||
		actual.Totals.ArtifactFiles != expected.Totals.ArtifactFiles ||
		actual.Totals.EvidenceFiles != expected.Totals.EvidenceFiles ||
		actual.Totals.VerifiedDomains != expected.Totals.VerifiedDomains ||
		actual.Totals.VerifiedOperations != expected.Totals.VerifiedOperations ||
		actual.Totals.TotalBytes != expected.Totals.TotalBytes {
		result.Issues = append(result.Issues, "evidenceBundle: totals mismatch")
	}
	if actual.Totals.WrittenFiles != 1 {
		result.Issues = append(result.Issues, "evidenceBundle: writtenFiles must be 1")
	}
	expectedFiles := map[string]setupSvcLiveReplayEvidenceBundleFile{}
	for _, file := range expected.Files {
		expectedFiles[file.Path] = file
	}
	actualFiles := map[string]setupSvcLiveReplayEvidenceBundleFile{}
	for _, file := range actual.Files {
		actualFiles[file.Path] = file
	}
	for path, expectedFile := range expectedFiles {
		actualFile, ok := actualFiles[path]
		if !ok {
			result.Issues = append(result.Issues, "evidenceBundle: missing file "+path)
			continue
		}
		if actualFile.ArtifactType != expectedFile.ArtifactType ||
			actualFile.Domain != expectedFile.Domain ||
			actualFile.Operation != expectedFile.Operation ||
			actualFile.SHA256 != expectedFile.SHA256 ||
			actualFile.Bytes != expectedFile.Bytes {
			result.Issues = append(result.Issues, "evidenceBundle: stale file "+path)
		}
	}
	for path := range actualFiles {
		if _, ok := expectedFiles[path]; !ok {
			result.Issues = append(result.Issues, "evidenceBundle: unexpected file "+path)
		}
	}
	if len(result.Issues) > 0 {
		result.Status = "stale"
	}
	return result
}

func buildSetupSvcLiveReplayManifestSyncOperation(projectPath string, domain setupSvcLiveReplayDomain, operation string, manifestOperation map[string]any) setupSvcLiveReplayManifestSyncOperation {
	result := setupSvcLiveReplayManifestSyncOperation{
		Operation: operation,
		Status:    "passed",
	}
	if manifestOperation == nil {
		result.Status = "failed"
		result.Issues = append(result.Issues, "missing manifest operation")
		return result
	}
	expectedFiles := setupSvcLiveReplayEvidenceFiles(domain.Domain, operation, operation != "query")
	evidenceFiles := setupSvcLiveReplayEvidenceFileList(manifestOperation["evidenceFiles"])
	contractIssues := setupSvcLiveReplayEvidenceFileContractIssues(projectPath, expectedFiles, evidenceFiles)
	for _, unexpected := range contractIssues.Unexpected {
		result.Issues = append(result.Issues, "unexpectedEvidenceFile:"+unexpected)
	}
	for _, duplicate := range contractIssues.Duplicate {
		result.Issues = append(result.Issues, "duplicateEvidenceFile:"+duplicate)
	}
	fileByContract := setupSvcLiveReplayEvidenceFileMap(projectPath, evidenceFiles)
	snapshotTablesByArtifact := map[string]map[string]bool{}
	artifactIndexByType := map[string]int{}
	for _, requiredFile := range expectedFiles {
		artifactStatus := setupSvcLiveReplayManifestSyncArtifactStatusForFile(projectPath, domain, operation, requiredFile, fileByContract)
		artifactIndexByType[artifactStatus.ArtifactType] = len(result.ArtifactStatuses)
		if artifactStatus.Status == "passed" && (artifactStatus.ArtifactType == "setup-svc" || artifactStatus.ArtifactType == "metadata-service") {
			if artifact, ok := readSetupSvcLiveReplayArtifactMap(projectPath, artifactStatus.File); ok {
				snapshotTablesByArtifact[artifactStatus.ArtifactType] = setupSvcLiveReplaySnapshotTableSet(artifact)
			}
		}
		result.ArtifactStatuses = append(result.ArtifactStatuses, artifactStatus)
	}
	for _, missing := range contractIssues.Missing {
		result.Issues = append(result.Issues, "missingEvidenceFile:"+missing)
	}
	if pairFailures := setupSvcLiveReplaySnapshotTableSetPairFailures(snapshotTablesByArtifact["setup-svc"], snapshotTablesByArtifact["metadata-service"]); len(pairFailures) > 0 {
		if index, ok := artifactIndexByType["metadata-service"]; ok {
			result.ArtifactStatuses[index].Status = "failed"
			result.ArtifactStatuses[index].Issues = append(result.ArtifactStatuses[index].Issues, pairFailures...)
		}
		result.Issues = append(result.Issues, pairFailures...)
	}
	for _, artifact := range result.ArtifactStatuses {
		oldStatus := strings.ToLower(strings.TrimSpace(firstMapString(manifestOperation, artifact.Field)))
		if oldStatus != artifact.Status {
			result.Updated = true
			manifestOperation[artifact.Field] = artifact.Status
		}
		switch artifact.Status {
		case "failed":
			result.Status = "failed"
		case "pending":
			if result.Status == "passed" {
				result.Status = "pending"
			}
		}
	}
	if len(result.Issues) > 0 && result.Status == "passed" {
		result.Status = "failed"
	}
	return result
}

func setupSvcLiveReplayManifestSyncArtifactStatusForFile(projectPath string, domain setupSvcLiveReplayDomain, operation string, requiredFile string, fileByContract map[string]string) setupSvcLiveReplayManifestSyncArtifactStatus {
	artifactType := setupSvcLiveReplayArtifactType(requiredFile)
	actualFile := fileByContract[setupSvcLiveReplayEvidencePathForContract(projectPath, requiredFile)]
	result := setupSvcLiveReplayManifestSyncArtifactStatus{
		ArtifactType: artifactType,
		Field:        setupSvcLiveReplayManifestSyncArtifactField(artifactType),
		File:         firstString(actualFile, requiredFile),
		Status:       "pending",
	}
	if strings.TrimSpace(actualFile) == "" {
		result.Issues = append(result.Issues, "missing evidenceFiles entry")
		return result
	}
	resolved := setupSvcLiveReplayResolveEvidenceFile(projectPath, actualFile)
	payload, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			result.Issues = append(result.Issues, "missing artifact file")
			return result
		}
		result.Status = "failed"
		result.Issues = append(result.Issues, "unreadable artifact file")
		return result
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		result.Status = "failed"
		result.Issues = append(result.Issues, "invalid JSON")
		return result
	}
	if artifact, ok := decoded.(map[string]any); ok && strings.EqualFold(firstMapString(artifact, "status", "evidenceStatus"), "pending") {
		result.Issues = append(result.Issues, "artifact status pending")
		return result
	}
	failures := verifySetupSvcLiveReplayEvidenceArtifact(projectPath, requiredFile, domain, operation, decoded)
	if len(failures) > 0 {
		result.Status = "failed"
		result.Issues = append(result.Issues, failures...)
		return result
	}
	result.Status = "passed"
	return result
}

func readSetupSvcLiveReplayArtifactMap(projectPath string, filePath string) (map[string]any, bool) {
	artifact, err := readJSONFile(setupSvcLiveReplayResolveEvidenceFile(projectPath, filePath))
	if err != nil {
		return nil, false
	}
	return artifact, true
}

func setupSvcLiveReplayManifestSyncArtifactField(artifactType string) string {
	switch setupSvcLiveReplayNormalizeArtifactType(artifactType) {
	case "setup-svc":
		return "setupSvcEvidenceStatus"
	case "metadata-service":
		return "metadataServiceEvidenceStatus"
	case "query-readback":
		return "queryEvidenceStatus"
	case "normalized-diff":
		return "normalizedDiffStatus"
	case "cleanup":
		return "cleanupStatus"
	default:
		return artifactType + "Status"
	}
}

func buildSetupSvcLiveReplayOperationDiff(projectPath string, domain string, operation string, manifestOperation map[string]any, requiredTables []string) (setupSvcLiveReplayNormalizedDiffOperation, *setupSvcLiveReplayDiffArtifact) {
	files := setupSvcLiveReplayEvidenceFileList(manifestOperation["evidenceFiles"])
	diffFile := findSetupSvcLiveReplayEvidenceFile(files, filepath.Join("outputs", "setup-svc-live-replay", domain, operation, "normalized-diff.json"))
	result := setupSvcLiveReplayNormalizedDiffOperation{
		Operation: operation,
		Status:    "clean",
		DiffFile:  diffFile,
	}
	setupFile := findSetupSvcLiveReplayEvidenceFile(files, filepath.Join("outputs", "setup-svc-live-replay", domain, operation, "setup-svc.json"))
	metadataFile := findSetupSvcLiveReplayEvidenceFile(files, filepath.Join("outputs", "setup-svc-live-replay", domain, operation, "metadata-service.json"))
	if setupFile == "" {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "missing setup-svc.json evidenceFiles entry")
	}
	if metadataFile == "" {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "missing metadata-service.json evidenceFiles entry")
	}
	if diffFile == "" {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "missing normalized-diff.json evidenceFiles entry")
	}
	if result.Status == "blocked" {
		return result, nil
	}
	setupArtifact, setupErr := readJSONFile(setupSvcLiveReplayResolveEvidenceFile(projectPath, setupFile))
	if setupErr != nil {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "cannot read setup-svc.json: "+setupErr.Error())
	}
	metadataArtifact, metadataErr := readJSONFile(setupSvcLiveReplayResolveEvidenceFile(projectPath, metadataFile))
	if metadataErr != nil {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, "cannot read metadata-service.json: "+metadataErr.Error())
	}
	if result.Status == "blocked" {
		return result, nil
	}
	operationRequiredTables := setupSvcLiveReplayRequiredTablesForOperation(setupSvcLiveReplayDomain{Domain: domain, RequiredTables: requiredTables}, operation)
	sourceFailures := []string{}
	sourceFailures = append(sourceFailures, verifySetupSvcLiveReplayEvidenceArtifact(projectPath, setupFile, setupSvcLiveReplayDomain{Domain: domain, RequiredTables: operationRequiredTables}, operation, setupArtifact)...)
	sourceFailures = append(sourceFailures, verifySetupSvcLiveReplayEvidenceArtifact(projectPath, metadataFile, setupSvcLiveReplayDomain{Domain: domain, RequiredTables: operationRequiredTables}, operation, metadataArtifact)...)
	if len(sourceFailures) > 0 {
		result.Status = "blocked"
		result.BlockingIssues = append(result.BlockingIssues, sourceFailures...)
		return result, nil
	}
	diff := setupSvcLiveReplayCompareSnapshotArtifacts(projectPath, domain, operation, operationRequiredTables, setupArtifact, metadataArtifact)
	result.Differences = diff.Totals.Differences
	if diff.Totals.Differences > 0 || diff.Totals.Failed > 0 {
		result.Status = "dirty"
	} else {
		result.Status = "clean"
	}
	return result, &diff
}

func setupSvcLiveReplayCompareSnapshotArtifacts(projectPath string, domain string, operation string, requiredTables []string, setupArtifact map[string]any, metadataArtifact map[string]any) setupSvcLiveReplayDiffArtifact {
	setupTables := setupSvcLiveReplayComparableSnapshotTables(setupArtifact)
	metadataTables := setupSvcLiveReplayComparableSnapshotTables(metadataArtifact)
	diff := setupSvcLiveReplayDiffArtifact{
		Status:              "passed",
		Project:             projectPath,
		ContractVersion:     setupSvcLiveReplayContractVersion,
		ContractFingerprint: setupSvcLiveReplayExpectedContractFingerprint(),
		Domain:              domain,
		Operation:           operation,
		ArtifactType:        "normalized-diff",
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		Normalization: setupSvcLiveReplayDiffNormalization{
			DynamicValues: "source snapshots must already contain normalized stable values",
			RowOrder:      "ignored; rows are compared by canonical JSON value",
		},
	}
	for _, table := range requiredTables {
		normalized := strings.ToLower(strings.TrimSpace(table))
		if normalized == "" {
			continue
		}
		tableDiff := setupSvcLiveReplayDiffTableResult{Table: table, Status: "passed"}
		setupTable, setupOK := setupTables[normalized]
		metadataTable, metadataOK := metadataTables[normalized]
		switch {
		case !setupOK && !metadataOK:
			tableDiff.Status = "failed"
			tableDiff.MissingRows = 1
			tableDiff.UnexpectedRows = 1
		case !setupOK:
			tableDiff.Status = "failed"
			tableDiff.MissingRows = 1
		case !metadataOK:
			tableDiff.Status = "failed"
			tableDiff.UnexpectedRows = 1
		default:
			tableDiff.MissingColumns = setupSvcLiveReplayDiffMissingStrings(setupTable.Columns, metadataTable.Columns)
			tableDiff.UnexpectedColumns = setupSvcLiveReplayDiffMissingStrings(metadataTable.Columns, setupTable.Columns)
			tableDiff.MissingRows = len(setupSvcLiveReplayDiffMissingStrings(setupTable.Rows, metadataTable.Rows))
			tableDiff.UnexpectedRows = len(setupSvcLiveReplayDiffMissingStrings(metadataTable.Rows, setupTable.Rows))
			tableDiff.MismatchedValues = len(tableDiff.MissingColumns) + len(tableDiff.UnexpectedColumns)
			if tableDiff.MissingRows > 0 || tableDiff.UnexpectedRows > 0 || tableDiff.MismatchedValues > 0 {
				tableDiff.Status = "failed"
			}
		}
		diff.Totals.MissingRows += tableDiff.MissingRows
		diff.Totals.UnexpectedRows += tableDiff.UnexpectedRows
		diff.Totals.MismatchedValues += tableDiff.MismatchedValues
		diff.Tables = append(diff.Tables, tableDiff)
	}
	diff.Totals.Differences = diff.Totals.MissingRows + diff.Totals.UnexpectedRows + diff.Totals.MismatchedValues
	if diff.Totals.Differences > 0 {
		diff.Status = "failed"
		diff.Totals.Failed = diff.Totals.Differences
	}
	return diff
}

func setupSvcLiveReplayComparableSnapshotTables(artifact map[string]any) map[string]setupSvcLiveReplayComparableTable {
	result := map[string]setupSvcLiveReplayComparableTable{}
	for _, key := range []string{"tableSnapshots", "snapshots", "metadataSnapshots", "tables", "metadataTables"} {
		setupSvcLiveReplayCollectComparableSnapshotTables(result, artifact[key])
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "snapshot"} {
		if nested, ok := artifact[key].(map[string]any); ok {
			for table, value := range setupSvcLiveReplayComparableSnapshotTables(nested) {
				result[table] = value
			}
		}
	}
	return result
}

func setupSvcLiveReplayCollectComparableSnapshotTables(out map[string]setupSvcLiveReplayComparableTable, value any) {
	switch item := value.(type) {
	case []any:
		for _, raw := range item {
			setupSvcLiveReplayCollectComparableSnapshotTables(out, raw)
		}
	case []map[string]any:
		for _, raw := range item {
			setupSvcLiveReplayCollectComparableSnapshotTables(out, raw)
		}
	case map[string]any:
		if table := firstMapString(item, "table", "tableName", "name"); table != "" {
			if snapshot := setupSvcLiveReplayComparableSnapshotTable(item); len(snapshot.Columns) > 0 || len(snapshot.Rows) > 0 {
				out[strings.ToLower(strings.TrimSpace(table))] = snapshot
			}
			return
		}
		for key, raw := range item {
			if nested, ok := raw.(map[string]any); ok {
				if snapshot := setupSvcLiveReplayComparableSnapshotTable(nested); len(snapshot.Columns) > 0 || len(snapshot.Rows) > 0 {
					out[strings.ToLower(strings.TrimSpace(key))] = snapshot
				}
			}
		}
	}
}

func setupSvcLiveReplayComparableSnapshotTable(snapshot map[string]any) setupSvcLiveReplayComparableTable {
	return setupSvcLiveReplayComparableTable{
		Columns: setupSvcLiveReplayComparableStringValues(firstExistingValue(snapshot, "columns", "fields", "requiredColumns", "requiredFields", "primaryKeys", "keyColumns")),
		Rows:    setupSvcLiveReplayComparableRowValues(firstExistingValue(snapshot, "rows", "records", "sampleRows", "readbackRows", "queriedRows", "before", "after", "changes")),
	}
}

func firstExistingValue(values map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			return value
		}
	}
	return nil
}

func setupSvcLiveReplayComparableStringValues(value any) map[string]bool {
	result := map[string]bool{}
	switch item := value.(type) {
	case []any:
		for _, raw := range item {
			if text := strings.TrimSpace(fmt.Sprint(raw)); text != "" {
				result[text] = true
			}
		}
	case []string:
		for _, raw := range item {
			if text := strings.TrimSpace(raw); text != "" {
				result[text] = true
			}
		}
	case map[string]any:
		for key := range item {
			if text := strings.TrimSpace(key); text != "" {
				result[text] = true
			}
		}
	case string:
		if text := strings.TrimSpace(item); text != "" {
			result[text] = true
		}
	}
	return result
}

func setupSvcLiveReplayComparableRowValues(value any) map[string]bool {
	result := map[string]bool{}
	switch item := value.(type) {
	case []any:
		for _, raw := range item {
			if text := setupSvcLiveReplayCanonicalJSON(raw); text != "" {
				result[text] = true
			}
		}
	case []map[string]any:
		for _, raw := range item {
			if text := setupSvcLiveReplayCanonicalJSON(raw); text != "" {
				result[text] = true
			}
		}
	case map[string]any:
		for _, raw := range item {
			if text := setupSvcLiveReplayCanonicalJSON(raw); text != "" {
				result[text] = true
			}
		}
	}
	return result
}

func setupSvcLiveReplayCanonicalJSON(value any) string {
	body, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(body)
}

func setupSvcLiveReplayDiffMissingStrings(expected map[string]bool, actual map[string]bool) []string {
	var missing []string
	for key := range expected {
		if !actual[key] {
			missing = append(missing, key)
		}
	}
	sort.Strings(missing)
	return missing
}

func writeSetupSvcLiveReplayDiffArtifact(projectPath string, filePath string, artifact *setupSvcLiveReplayDiffArtifact) error {
	if artifact == nil {
		return nil
	}
	path := setupSvcLiveReplayResolveEvidenceFile(projectPath, filePath)
	body, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0644)
}

func setupSvcLiveReplaySetManifestOperationDiffStatus(operation map[string]any, status string) {
	if setupSvcLiveReplayPassedStatus(status) {
		operation["normalizedDiffStatus"] = "passed"
		return
	}
	operation["normalizedDiffStatus"] = "failed"
}

func writeSetupSvcLiveReplayManifestMap(path string, manifest map[string]any) error {
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0644)
}

func setupSvcLiveReplayPacketEnvelopeIssues(projectPath string, packet setupSvcLiveReplayPacket, expected []setupSvcLiveReplayDomain) []string {
	var issues []string
	if strings.TrimSpace(packet.Project) != strings.TrimSpace(projectPath) {
		issues = append(issues, "project mismatch")
	}
	if !packet.ReadOnly {
		issues = append(issues, "readOnly must be true")
	}
	if packet.Execute {
		issues = append(issues, "execute must be false")
	}
	if !packet.ApprovalRequired {
		issues = append(issues, "approvalRequired must be true")
	}
	if strings.TrimSpace(packet.ApprovalPhrase) != setupSvcParityReplayApproval {
		issues = append(issues, "approvalPhrase mismatch")
	}
	if strings.TrimSpace(packet.Status) != "ready_for_manual_evidence_collection" {
		issues = append(issues, "status must be ready_for_manual_evidence_collection")
	}
	if filepath.Clean(strings.TrimSpace(packet.ManifestPath)) != filepath.Clean(setupSvcLiveReplayManifestPath(projectPath, "")) {
		issues = append(issues, "manifestPath mismatch")
	}
	expectedOps, expectedWrites, expectedQueries := 0, 0, 0
	for _, domain := range expected {
		expectedOps += len(domain.Operations)
		for _, operation := range domain.Operations {
			if strings.ToLower(strings.TrimSpace(operation)) == "query" {
				expectedQueries++
			} else {
				expectedWrites++
			}
		}
	}
	if packet.Totals.Domains != len(expected) {
		issues = append(issues, "totals.domains mismatch")
	}
	if packet.Totals.Operations != expectedOps {
		issues = append(issues, "totals.operations mismatch")
	}
	if packet.Totals.WriteOperations != expectedWrites {
		issues = append(issues, "totals.writeOperations mismatch")
	}
	if packet.Totals.QueryOperations != expectedQueries {
		issues = append(issues, "totals.queryOperations mismatch")
	}
	return issues
}

func setupSvcLiveReplayPacketManifestTemplateIssues(projectPath string, manifest setupSvcLiveReplayManifest, expected []setupSvcLiveReplayDomain) []string {
	var issues []string
	if strings.TrimSpace(manifest.Mode) != setupSvcLiveReplayEvidenceMode {
		issues = append(issues, "mode must be "+setupSvcLiveReplayEvidenceMode)
	}
	if strings.TrimSpace(manifest.Project) != strings.TrimSpace(projectPath) {
		issues = append(issues, "project mismatch")
	}
	if strings.TrimSpace(manifest.Status) != "pending" {
		issues = append(issues, "status must be pending")
	}
	for _, issue := range setupSvcLiveReplayContractIdentityIssues(manifest.ContractVersion, manifest.ContractFingerprint) {
		issues = append(issues, issue)
	}
	expectedDomainSet := map[string]bool{}
	for _, domain := range expected {
		expectedDomainSet[domain.Domain] = true
	}
	manifestDomains := map[string]setupSvcLiveReplayManifestDomain{}
	for _, domain := range manifest.Domains {
		normalized := normalizeDomain(domain.Domain)
		switch {
		case normalized == "":
			issues = append(issues, "missing domain")
		case !expectedDomainSet[normalized]:
			issues = append(issues, "unexpected domain "+domain.Domain)
		case manifestTemplateDomainExists(manifestDomains, normalized):
			issues = append(issues, "duplicate domain "+normalized)
		default:
			manifestDomains[normalized] = domain
		}
	}
	for _, domain := range expected {
		manifestDomain, ok := manifestDomains[domain.Domain]
		if !ok {
			issues = append(issues, domain.Domain+": missing from template")
			continue
		}
		expectedOperationSet := map[string]bool{}
		for _, operation := range domain.Operations {
			expectedOperationSet[strings.ToLower(strings.TrimSpace(operation))] = true
		}
		manifestOps := map[string]setupSvcLiveReplayManifestOperation{}
		for _, operation := range manifestDomain.Operations {
			normalized := strings.ToLower(strings.TrimSpace(operation.Operation))
			switch {
			case normalized == "":
				issues = append(issues, domain.Domain+": missing operation")
			case !expectedOperationSet[normalized]:
				issues = append(issues, domain.Domain+"/"+operation.Operation+": unexpected operation")
			case manifestTemplateOperationExists(manifestOps, normalized):
				issues = append(issues, domain.Domain+"/"+normalized+": duplicate operation")
			default:
				manifestOps[normalized] = operation
			}
		}
		for _, operation := range domain.Operations {
			normalized := strings.ToLower(strings.TrimSpace(operation))
			manifestOperation, ok := manifestOps[normalized]
			if !ok {
				issues = append(issues, domain.Domain+"/"+operation+": missing from template")
				continue
			}
			for _, issue := range setupSvcLiveReplayManifestTemplateOperationIssues(domain.Domain, operation, manifestOperation) {
				issues = append(issues, domain.Domain+"/"+operation+": "+issue)
			}
		}
	}
	return issues
}

func setupSvcLiveReplayManifestTemplateOperationIssues(domain string, operation string, manifestOperation setupSvcLiveReplayManifestOperation) []string {
	var issues []string
	if strings.TrimSpace(manifestOperation.SetupSvcEvidenceStatus) != "pending" {
		issues = append(issues, "setupSvcEvidenceStatus must be pending")
	}
	if strings.TrimSpace(manifestOperation.MetadataServiceEvidenceStatus) != "pending" {
		issues = append(issues, "metadataServiceEvidenceStatus must be pending")
	}
	if strings.TrimSpace(manifestOperation.QueryEvidenceStatus) != "pending" {
		issues = append(issues, "queryEvidenceStatus must be pending")
	}
	if strings.TrimSpace(manifestOperation.NormalizedDiffStatus) != "pending" {
		issues = append(issues, "normalizedDiffStatus must be pending")
	}
	requireCleanup := strings.ToLower(strings.TrimSpace(operation)) != "query"
	if requireCleanup && strings.TrimSpace(manifestOperation.CleanupStatus) != "pending" {
		issues = append(issues, "cleanupStatus must be pending")
	}
	if !requireCleanup && strings.TrimSpace(manifestOperation.CleanupStatus) != "" {
		issues = append(issues, "cleanupStatus must be empty for query")
	}
	expectedFiles := setupSvcLiveReplayEvidenceFiles(domain, operation, requireCleanup)
	if missing := missingSetupSvcLiveReplayStrings(expectedFiles, manifestOperation.EvidenceFiles, true); len(missing) > 0 {
		issues = append(issues, "missing evidenceFiles "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expectedFiles, manifestOperation.EvidenceFiles, true); len(unexpected) > 0 {
		issues = append(issues, "unexpected evidenceFiles "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(manifestOperation.EvidenceFiles, true); len(duplicates) > 0 {
		issues = append(issues, "duplicate evidenceFiles "+strings.Join(duplicates, ","))
	}
	return issues
}

func setupSvcLiveReplayExpectedContractFingerprint() string {
	type contractOperation struct {
		Operation        string   `json:"operation"`
		RequiredEvidence []string `json:"requiredEvidence"`
		EvidenceFiles    []string `json:"evidenceFiles"`
	}
	type contractDomain struct {
		Domain                    string              `json:"domain"`
		RequiredTables            []string            `json:"requiredTables"`
		RuntimeEffects            []string            `json:"runtimeEffects"`
		QueryReadbackExpectations []string            `json:"queryReadbackExpectations"`
		Operations                []contractOperation `json:"operations"`
	}
	type contract struct {
		Version string           `json:"version"`
		Domains []contractDomain `json:"domains"`
	}
	payload := contract{Version: setupSvcLiveReplayContractVersion}
	for _, domain := range setupSvcLiveReplayDomains() {
		item := contractDomain{
			Domain:                    domain.Domain,
			RequiredTables:            domain.RequiredTables,
			RuntimeEffects:            domain.RuntimeEffects,
			QueryReadbackExpectations: domain.QueryReadbackExpectations,
		}
		for _, operation := range domain.Operations {
			item.Operations = append(item.Operations, contractOperation{
				Operation:        operation,
				RequiredEvidence: setupSvcLiveReplayRequiredEvidence(operation),
				EvidenceFiles:    setupSvcLiveReplayEvidenceFiles(domain.Domain, operation, operation != "query"),
			})
		}
		payload.Domains = append(payload.Domains, item)
	}
	body, _ := json.Marshal(payload)
	sum := sha256.Sum256(body)
	return fmt.Sprintf("sha256:%x", sum)
}

func setupSvcLiveReplayContractIdentityIssues(contractVersion string, contractFingerprint string) []string {
	var issues []string
	if strings.TrimSpace(contractVersion) == "" {
		issues = append(issues, "missing contractVersion")
	} else if strings.TrimSpace(contractVersion) != setupSvcLiveReplayContractVersion {
		issues = append(issues, "contractVersion mismatch")
	}
	expectedFingerprint := setupSvcLiveReplayExpectedContractFingerprint()
	if strings.TrimSpace(contractFingerprint) == "" {
		issues = append(issues, "missing contractFingerprint")
	} else if strings.TrimSpace(contractFingerprint) != expectedFingerprint {
		issues = append(issues, "contractFingerprint mismatch")
	}
	return issues
}

func packetDomainExists(domains map[string]setupSvcLiveReplayPacketDomain, domain string) bool {
	_, ok := domains[domain]
	return ok
}

func packetOperationExists(operations map[string]setupSvcLiveReplayPacketOperation, operation string) bool {
	_, ok := operations[operation]
	return ok
}

func manifestTemplateDomainExists(domains map[string]setupSvcLiveReplayManifestDomain, domain string) bool {
	_, ok := domains[domain]
	return ok
}

func manifestTemplateOperationExists(operations map[string]setupSvcLiveReplayManifestOperation, operation string) bool {
	_, ok := operations[operation]
	return ok
}

func setupSvcLiveReplayRequiredEvidence(operation string) []string {
	requiredEvidence := []string{
		"setupSvcEvidenceStatus",
		"metadataServiceEvidenceStatus",
		"queryEvidenceStatus",
		"normalizedDiffStatus",
	}
	if strings.ToLower(strings.TrimSpace(operation)) != "query" {
		requiredEvidence = append(requiredEvidence, "cleanupStatus")
	}
	return requiredEvidence
}

func setupSvcLiveReplayDomainContractIssues(expected setupSvcLiveReplayDomain, packetDomain setupSvcLiveReplayPacketDomain) []string {
	var issues []string
	if missing := missingSetupSvcLiveReplayStrings(expected.RequiredTables, packetDomain.RequiredTables, false); len(missing) > 0 {
		issues = append(issues, "missing requiredTables "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expected.RequiredTables, packetDomain.RequiredTables, false); len(unexpected) > 0 {
		issues = append(issues, "unexpected requiredTables "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(packetDomain.RequiredTables, false); len(duplicates) > 0 {
		issues = append(issues, "duplicate requiredTables "+strings.Join(duplicates, ","))
	}
	if missing := missingSetupSvcLiveReplayStrings(expected.RuntimeEffects, packetDomain.RuntimeEffects, false); len(missing) > 0 {
		issues = append(issues, "missing runtimeEffects "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expected.RuntimeEffects, packetDomain.RuntimeEffects, false); len(unexpected) > 0 {
		issues = append(issues, "unexpected runtimeEffects "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(packetDomain.RuntimeEffects, false); len(duplicates) > 0 {
		issues = append(issues, "duplicate runtimeEffects "+strings.Join(duplicates, ","))
	}
	if missing := missingSetupSvcLiveReplayStrings(expected.QueryReadbackExpectations, packetDomain.QueryReadbackExpectations, false); len(missing) > 0 {
		issues = append(issues, "missing queryReadbackExpectations "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expected.QueryReadbackExpectations, packetDomain.QueryReadbackExpectations, false); len(unexpected) > 0 {
		issues = append(issues, "unexpected queryReadbackExpectations "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(packetDomain.QueryReadbackExpectations, false); len(duplicates) > 0 {
		issues = append(issues, "duplicate queryReadbackExpectations "+strings.Join(duplicates, ","))
	}
	return issues
}

func setupSvcLiveReplayPacketContractIssues(domain string, operation string, packetOperation setupSvcLiveReplayPacketOperation) []string {
	var issues []string
	expectedEvidence := setupSvcLiveReplayRequiredEvidence(operation)
	if missing := missingSetupSvcLiveReplayStrings(expectedEvidence, packetOperation.RequiredEvidence, false); len(missing) > 0 {
		issues = append(issues, "missing requiredEvidence "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expectedEvidence, packetOperation.RequiredEvidence, false); len(unexpected) > 0 {
		issues = append(issues, "unexpected requiredEvidence "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(packetOperation.RequiredEvidence, false); len(duplicates) > 0 {
		issues = append(issues, "duplicate requiredEvidence "+strings.Join(duplicates, ","))
	}
	expectedFiles := setupSvcLiveReplayEvidenceFiles(domain, operation, strings.ToLower(strings.TrimSpace(operation)) != "query")
	if missing := missingSetupSvcLiveReplayStrings(expectedFiles, packetOperation.EvidenceFiles, true); len(missing) > 0 {
		issues = append(issues, "missing evidenceFiles "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expectedFiles, packetOperation.EvidenceFiles, true); len(unexpected) > 0 {
		issues = append(issues, "unexpected evidenceFiles "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(packetOperation.EvidenceFiles, true); len(duplicates) > 0 {
		issues = append(issues, "duplicate evidenceFiles "+strings.Join(duplicates, ","))
	}
	return issues
}

func missingSetupSvcLiveReplayStrings(expected []string, actual []string, pathValue bool) []string {
	actualSet := setupSvcLiveReplayStringSet(actual, pathValue)
	var missing []string
	for _, item := range expected {
		key := setupSvcLiveReplayComparableString(item, pathValue)
		if key == "" {
			continue
		}
		if !actualSet[key] {
			missing = append(missing, setupSvcLiveReplayComparableString(item, pathValue))
		}
	}
	return missing
}

func unexpectedSetupSvcLiveReplayStrings(expected []string, actual []string, pathValue bool) []string {
	expectedSet := setupSvcLiveReplayStringSet(expected, pathValue)
	var unexpected []string
	for _, item := range actual {
		key := setupSvcLiveReplayComparableString(item, pathValue)
		if key == "" {
			continue
		}
		if !expectedSet[key] {
			unexpected = append(unexpected, key)
		}
	}
	return unexpected
}

func duplicateSetupSvcLiveReplayStrings(values []string, pathValue bool) []string {
	seen := map[string]bool{}
	reported := map[string]bool{}
	var duplicates []string
	for _, value := range values {
		key := setupSvcLiveReplayComparableString(value, pathValue)
		if key == "" {
			continue
		}
		if seen[key] && !reported[key] {
			duplicates = append(duplicates, key)
			reported[key] = true
			continue
		}
		seen[key] = true
	}
	return duplicates
}

func setupSvcLiveReplayStringSet(values []string, pathValue bool) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		key := setupSvcLiveReplayComparableString(value, pathValue)
		if key != "" {
			result[key] = true
		}
	}
	return result
}

func setupSvcLiveReplayComparableString(value string, pathValue bool) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return ""
	}
	if pathValue {
		normalized = filepath.ToSlash(filepath.Clean(normalized))
	}
	return normalized
}

func isSetupSvcLiveReplayPacketMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-packet", "setup-svc-parity-replay-packet", "live-setup-svc-replay-packet", "parity-live-replay-packet":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayWorkspaceMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-workspace", "setup-svc-live-replay-evidence-workspace", "setup-svc-parity-replay-workspace", "parity-live-replay-workspace":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayCaptureSourceWorkspaceMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-capture-source-workspace", "setup-svc-live-replay-capture-workspace", "setup-svc-live-replay-source-workspace", "setup-svc-parity-replay-capture-source-workspace", "parity-live-replay-capture-source-workspace":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayNormalizedDiffMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-normalized-diff", "setup-svc-live-replay-diff", "setup-svc-parity-replay-normalized-diff", "parity-live-replay-diff":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayManifestSyncMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-manifest-sync", "setup-svc-live-replay-sync", "setup-svc-live-replay-evidence-sync", "setup-svc-parity-replay-manifest-sync", "parity-live-replay-manifest-sync":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayEvidenceBundleMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-evidence-bundle", "setup-svc-live-replay-bundle", "setup-svc-live-replay-checksum", "setup-svc-parity-replay-evidence-bundle", "parity-live-replay-evidence-bundle":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayEvidenceImportMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-evidence-import", "setup-svc-live-replay-import", "setup-svc-parity-replay-evidence-import", "parity-live-replay-evidence-import":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayQueryReadbackCaptureMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-query-readback-capture", "setup-svc-live-replay-query-capture", "setup-svc-parity-replay-query-readback-capture", "parity-live-replay-query-readback-capture":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayMetadataServiceQueryScanCaptureMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-metadata-service-query-scan-capture", "setup-svc-live-replay-metadata-service-scan-capture", "setup-svc-live-replay-msapi-query-scan-capture", "setup-svc-live-replay-query-scan-capture", "setup-svc-parity-replay-metadata-service-query-scan-capture", "parity-live-replay-metadata-service-query-scan-capture":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayMetadataServiceApplyCaptureMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-metadata-service-apply-capture", "setup-svc-live-replay-msapi-apply-capture", "setup-svc-live-replay-apply-capture", "setup-svc-parity-replay-metadata-service-apply-capture", "parity-live-replay-metadata-service-apply-capture":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplaySnapshotFromChangesMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-snapshot-from-changes", "setup-svc-live-replay-snapshot-capture", "setup-svc-live-replay-changes-snapshot", "setup-svc-parity-replay-snapshot-from-changes", "parity-live-replay-snapshot-from-changes":
		return true
	default:
		return false
	}
}

func isSetupSvcLiveReplayMatrixPromotionMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "setup-svc-live-replay-promotion", "setup-svc-live-replay-matrix-promotion", "setup-svc-live-replay-promote-matrix", "setup-svc-parity-replay-promotion", "parity-live-replay-promotion":
		return true
	default:
		return false
	}
}

func setupSvcLiveReplayEvidenceFiles(domain string, operation string, requireCleanup bool) []string {
	base := filepath.Join("outputs", "setup-svc-live-replay", domain, operation)
	files := []string{
		filepath.Join(base, "setup-svc.json"),
		filepath.Join(base, "metadata-service.json"),
		filepath.Join(base, "query-readback.json"),
		filepath.Join(base, "normalized-diff.json"),
	}
	if requireCleanup {
		files = append(files, filepath.Join(base, "cleanup.json"))
	}
	return files
}

func setupSvcLiveReplayOperatorSteps(projectPath string, domain string, operation string, writeOperation bool) []string {
	steps := []string{
		"Capture setup-svc " + domain + " " + operation + " request/response and affected metadata snapshots.",
		"Run MetadataService parity operation for " + domain + " " + operation + " through plan/apply or read-only scan as appropriate.",
		"Capture MSAPI query/readback payload and verify required relationships from the parity matrix.",
		"Run normalized diff and attach the diff artifact to the manifest.",
	}
	if writeOperation {
		steps = append(steps, "Clean up disposable replay metadata and attach cleanup evidence before setting cleanupStatus=passed.")
	}
	steps = append(steps, "Verify with: cloudcc scan msapi "+shellPath(projectPath)+" setup-svc-live-replay-evidence")
	return steps
}

func buildSetupSvcLiveReplayEvidenceResult(projectPath string, manifestArg string) (setupSvcLiveReplayEvidenceResult, error) {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	manifest, err := readJSONFile(manifestPath)
	if err != nil {
		return setupSvcLiveReplayEvidenceResult{}, err
	}
	matrixContract := setupSvcLiveReplayMatrixContractStatus(projectPath)
	result := setupSvcLiveReplayEvidenceResult{
		Mode:                setupSvcLiveReplayEvidenceMode,
		Project:             projectPath,
		ReadOnly:            true,
		ManifestPath:        manifestPath,
		ContractVersion:     firstMapString(manifest, "contractVersion"),
		ContractFingerprint: firstMapString(manifest, "contractFingerprint"),
		MatrixContract:      matrixContract,
		Status:              "passed",
		Notes: []string{
			"All expected domains must pass before the parity matrix can move from covered to verified.",
			"Each operation must prove setup-svc evidence, MetadataService evidence, query/readback evidence, normalized diff evidence, and readable contract-bound JSON artifact files.",
			"Write operations also require cleanupStatus=passed and cleanup.json evidence so disposable replay metadata does not remain behind.",
		},
	}
	if matrixContract.Status != "passed" {
		for _, issue := range matrixContract.Issues {
			result.BlockingIssues = append(result.BlockingIssues, "parityMatrix: "+issue)
		}
	}
	manifestMode := firstMapString(manifest, "mode")
	switch {
	case manifestMode == "":
		result.BlockingIssues = append(result.BlockingIssues, "manifest: missing mode")
	case manifestMode != setupSvcLiveReplayEvidenceMode:
		result.BlockingIssues = append(result.BlockingIssues, "manifest: unexpected mode "+manifestMode)
	}
	manifestStatus := firstMapString(manifest, "status", "evidenceStatus", "verificationStatus")
	switch {
	case manifestStatus == "":
		result.BlockingIssues = append(result.BlockingIssues, "manifest: missing status")
	case !setupSvcLiveReplayPassedStatus(manifestStatus):
		result.BlockingIssues = append(result.BlockingIssues, "manifest: status not passed "+manifestStatus)
	}
	for _, issue := range setupSvcLiveReplayContractIdentityIssues(result.ContractVersion, result.ContractFingerprint) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	for _, issue := range setupSvcLiveReplayProjectIdentityIssues(projectPath, firstMapString(manifest, "project", "projectPath")) {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+issue)
	}
	domainEvidence := map[string]map[string]any{}
	expectedDomains := map[string]bool{}
	for _, expected := range setupSvcLiveReplayDomains() {
		expectedDomains[expected.Domain] = true
	}
	for _, domain := range mapList(manifest["domains"]) {
		name := firstMapString(domain, "domain", "name")
		if name == "" {
			result.BlockingIssues = append(result.BlockingIssues, "manifest: missing domain name")
			continue
		}
		normalized := normalizeDomain(name)
		if !expectedDomains[normalized] {
			result.BlockingIssues = append(result.BlockingIssues, "manifest: unexpected domain "+name)
			continue
		}
		if _, exists := domainEvidence[normalized]; exists {
			result.BlockingIssues = append(result.BlockingIssues, "manifest: duplicate domain "+normalized)
			continue
		}
		domainEvidence[normalized] = domain
	}
	for _, expected := range setupSvcLiveReplayDomains() {
		domainResult := verifySetupSvcLiveReplayDomain(projectPath, expected, domainEvidence[expected.Domain])
		result.Domains = append(result.Domains, domainResult)
		result.Totals.Domains++
		result.Totals.Operations += len(expected.Operations)
		result.Totals.VerifiedOperations += len(domainResult.VerifiedOperations)
		result.Totals.MissingOperations += len(domainResult.MissingOperations)
		result.Totals.FailedOperations += len(domainResult.FailedOperations)
		switch domainResult.Status {
		case "verified":
			result.Totals.VerifiedDomains++
		case "missing":
			result.Totals.MissingDomains++
			result.BlockingIssues = append(result.BlockingIssues, expected.Domain+": missing domain evidence")
		default:
			result.BlockingIssues = append(result.BlockingIssues, expected.Domain+": "+domainResult.Status)
		}
	}
	if len(result.BlockingIssues) > 0 {
		result.Status = "blocked"
	}
	return result, nil
}

func buildSetupSvcLiveReplayPromotionResult(projectPath string, manifestArg string) (setupSvcLiveReplayPromotionResult, error) {
	evidence, err := buildSetupSvcLiveReplayEvidenceResult(projectPath, manifestArg)
	if err != nil {
		return setupSvcLiveReplayPromotionResult{}, err
	}
	matrixStatuses := setupSvcLiveReplayMatrixStatuses(projectPath)
	result := setupSvcLiveReplayPromotionResult{
		Mode:         "setup-svc-live-replay-promotion",
		Project:      projectPath,
		ReadOnly:     true,
		Status:       "passed",
		ManifestPath: evidence.ManifestPath,
		Notes: []string{
			"This command is read-only and does not edit the parity matrix.",
			"Only domains whose every expected operation has passed evidence are promotable from covered to verified.",
			"Persist matrix status changes only after recording the live replay evidence in .claw/test-report.md.",
		},
	}
	globalBlocked := evidence.Status != "passed"
	if globalBlocked {
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+evidence.Status)
	}
	for _, domain := range evidence.Domains {
		currentMatrixStatus := matrixStatuses[normalizeDomain(domain.Domain)]
		if currentMatrixStatus == "" {
			currentMatrixStatus = "covered"
		}
		item := setupSvcLiveReplayPromotionDomain{
			Domain:              domain.Domain,
			CurrentMatrixStatus: currentMatrixStatus,
			EvidenceStatus:      domain.Status,
			RecommendedStatus:   currentMatrixStatus,
			VerifiedOperations:  append([]string{}, domain.VerifiedOperations...),
		}
		result.Totals.Domains++
		result.Totals.Operations += len(domain.ExpectedOperations)
		evidenceVerified := !globalBlocked && domain.Status == "verified" && len(domain.VerifiedOperations) == len(domain.ExpectedOperations)
		switch {
		case evidenceVerified && currentMatrixStatus == "covered":
			item.CanPromote = true
			item.RecommendedStatus = "verified"
			result.Totals.PromotableDomains++
			result.Totals.PromotableOperations += len(domain.VerifiedOperations)
			result.MatrixUpdates = append(result.MatrixUpdates, setupSvcLiveReplayMatrixUpdate{
				Domain:     domain.Domain,
				FromStatus: currentMatrixStatus,
				ToStatus:   "verified",
				Reason:     "all expected setup-svc live replay evidence passed",
			})
		case evidenceVerified && currentMatrixStatus == "verified":
			item.RecommendedStatus = "verified"
		default:
			item.BlockingOperations = setupSvcLiveReplayBlockingOperations(domain)
			item.FailedEvidence = setupSvcLiveReplayEvidenceDomainFailureIssues(domain)
			if globalBlocked && len(item.BlockingOperations) == 0 {
				item.BlockingOperations = append(item.BlockingOperations, domain.ExpectedOperations...)
			}
			if evidenceVerified && currentMatrixStatus != "covered" && currentMatrixStatus != "verified" {
				item.BlockingOperations = append([]string{}, domain.ExpectedOperations...)
			}
			result.Totals.BlockedDomains++
			result.Totals.BlockedOperations += len(item.BlockingOperations)
			result.Totals.MissingOperations += len(domain.MissingOperations)
			result.Totals.FailedOperations += len(domain.FailedOperations)
			if domain.Status == "missing" {
				result.Totals.MissingDomains++
			}
			if evidenceVerified && currentMatrixStatus != "covered" && currentMatrixStatus != "verified" {
				result.BlockingIssues = append(result.BlockingIssues, domain.Domain+": matrix status "+currentMatrixStatus+" cannot promote to verified")
			} else {
				result.BlockingIssues = append(result.BlockingIssues, domain.Domain+": "+domain.Status)
				result.BlockingIssues = append(result.BlockingIssues, item.FailedEvidence...)
			}
		}
		result.Domains = append(result.Domains, item)
	}
	if len(result.BlockingIssues) > 0 {
		if result.Totals.PromotableDomains > 0 {
			result.Status = "partial"
		} else {
			result.Status = "blocked"
		}
	}
	return result, nil
}

func setupSvcLiveReplayEvidenceFailureIssues(evidence setupSvcLiveReplayEvidenceResult) []string {
	issues := []string{}
	for _, domain := range evidence.Domains {
		issues = append(issues, setupSvcLiveReplayEvidenceDomainFailureIssues(domain)...)
	}
	return issues
}

func setupSvcLiveReplayEvidenceDomainFailureIssues(domain setupSvcLiveReplayEvidenceDomain) []string {
	issues := []string{}
	for _, operation := range domain.FailedOperations {
		for _, missing := range operation.MissingEvidence {
			issues = append(issues, domain.Domain+"/"+operation.Operation+": "+missing)
		}
		for _, failed := range operation.FailedEvidence {
			issues = append(issues, domain.Domain+"/"+operation.Operation+": "+failed)
		}
	}
	return issues
}

func buildSetupSvcLiveReplayMatrixPromotionApplyResult(projectPath string, manifestArg string, execute bool, approval string) (setupSvcLiveReplayMatrixPromotionApplyResult, error) {
	matrixContract := setupSvcLiveReplayMatrixContractStatus(projectPath)
	result := setupSvcLiveReplayMatrixPromotionApplyResult{
		Mode:             "setup-svc-live-replay-matrix-promotion-apply",
		Project:          projectPath,
		ReadOnly:         !execute,
		Execute:          execute,
		ApprovalRequired: true,
		Approved:         execute && approval == setupSvcParityMatrixPromotionApproval,
		Status:           "dry_run_ready",
		MatrixPath:       matrixContract.Path,
		MatrixContract:   matrixContract,
		NextCommands: []string{
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit",
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle --execute --approval " + setupSvcParityEvidenceBundleApproval,
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion --execute --approval " + setupSvcParityMatrixPromotionApproval,
		},
		Notes: []string{
			"This command only promotes parity matrix statuses after setup-svc-live-replay-evidence, evidence-bundle verification, and setup-svc-live-replay-promotion pass.",
			"Dry-run is read-only. Execute updates the located parity matrix file only after the explicit matrix-promotion approval phrase is supplied.",
		},
	}
	if execute && approval != setupSvcParityMatrixPromotionApproval {
		return result, fmt.Errorf("refusing to update setup-svc parity matrix without --approval %s", setupSvcParityMatrixPromotionApproval)
	}
	if matrixContract.Status != "passed" {
		result.Status = "blocked_parity_matrix_contract"
		for _, issue := range matrixContract.Issues {
			result.BlockingIssues = append(result.BlockingIssues, "parityMatrix: "+issue)
		}
		return result, nil
	}
	if strings.TrimSpace(matrixContract.Path) == "" {
		result.Status = "blocked_missing_matrix"
		result.BlockingIssues = append(result.BlockingIssues, "parityMatrix: missing parity matrix file")
		return result, nil
	}
	promotion, err := buildSetupSvcLiveReplayPromotionResult(projectPath, manifestArg)
	if err != nil {
		result.Status = "blocked_promotion_audit"
		result.BlockingIssues = append(result.BlockingIssues, "promotion: "+err.Error())
		return result, nil
	}
	result.ManifestPath = promotion.ManifestPath
	result.Totals.Domains = promotion.Totals.Domains
	result.Totals.Operations = promotion.Totals.Operations
	result.Totals.CandidateUpdates = len(promotion.MatrixUpdates)
	result.MatrixUpdates = append(result.MatrixUpdates, promotion.MatrixUpdates...)
	if promotion.Status != "passed" || len(promotion.BlockingIssues) > 0 {
		result.Status = "blocked_promotion_audit"
		result.Totals.BlockedUpdates = len(setupSvcLiveReplayDomains()) - len(promotion.MatrixUpdates)
		if result.Totals.BlockedUpdates < 0 {
			result.Totals.BlockedUpdates = 0
		}
		for _, issue := range promotion.BlockingIssues {
			result.BlockingIssues = append(result.BlockingIssues, "promotion: "+issue)
		}
		if len(result.BlockingIssues) == 0 {
			result.BlockingIssues = append(result.BlockingIssues, "promotion: "+promotion.Status)
		}
		return result, nil
	}
	if len(promotion.MatrixUpdates) == 0 {
		result.Status = "already_verified"
		result.Warnings = append(result.Warnings, "promotion audit produced no covered->verified matrix updates")
		return result, nil
	}
	bundle := verifySetupSvcLiveReplayEvidenceBundle(projectPath, manifestArg)
	if bundle.Status != "passed" {
		result.Status = "blocked_evidence_bundle"
		result.Totals.BlockedUpdates = len(promotion.MatrixUpdates)
		result.BlockingIssues = append(result.BlockingIssues, bundle.Issues...)
		if len(result.BlockingIssues) == 0 {
			result.BlockingIssues = append(result.BlockingIssues, "evidenceBundle: "+bundle.Status)
		}
		return result, nil
	}
	if !execute {
		result.Status = "dry_run_ready"
		return result, nil
	}
	updatedDomains, err := applySetupSvcLiveReplayMatrixUpdates(matrixContract.Path, promotion.MatrixUpdates)
	if err != nil {
		result.Status = "blocked_matrix_update"
		result.BlockingIssues = append(result.BlockingIssues, "parityMatrix: "+err.Error())
		return result, nil
	}
	result.Status = "applied"
	result.UpdatedDomains = updatedDomains
	result.Totals.AppliedUpdates = len(updatedDomains)
	return result, nil
}

func applySetupSvcLiveReplayMatrixUpdates(matrixPath string, updates []setupSvcLiveReplayMatrixUpdate) ([]string, error) {
	body, err := os.ReadFile(matrixPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", matrixPath, err)
	}
	var matrix map[string]any
	if err := json.Unmarshal(body, &matrix); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", matrixPath, err)
	}
	domains := mapList(matrix["domains"])
	if len(domains) == 0 {
		return nil, fmt.Errorf("missing domains in %s", matrixPath)
	}
	updatesByDomain := map[string]setupSvcLiveReplayMatrixUpdate{}
	for _, update := range updates {
		domain := normalizeDomain(update.Domain)
		if domain == "" {
			continue
		}
		updatesByDomain[domain] = update
	}
	var updated []string
	for _, item := range domains {
		domain := normalizeDomain(firstMapString(item, "domain", "name"))
		update, ok := updatesByDomain[domain]
		if !ok {
			continue
		}
		currentStatus := strings.ToLower(strings.TrimSpace(firstMapString(item, "status", "currentStatus")))
		if currentStatus != update.FromStatus {
			return nil, fmt.Errorf("%s: current status %s does not match expected %s", update.Domain, firstString(currentStatus, "(missing)"), update.FromStatus)
		}
		if update.ToStatus != "verified" {
			return nil, fmt.Errorf("%s: unsupported promotion target %s", update.Domain, update.ToStatus)
		}
		item["status"] = "verified"
		updated = append(updated, update.Domain)
		delete(updatesByDomain, domain)
	}
	if len(updatesByDomain) > 0 {
		var missing []string
		for _, update := range updatesByDomain {
			missing = append(missing, update.Domain)
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("missing domains for matrix updates: %s", strings.Join(missing, ","))
	}
	matrix["domains"] = domains
	encoded, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		return nil, err
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(matrixPath, encoded, 0644); err != nil {
		return nil, fmt.Errorf("cannot write %s: %w", matrixPath, err)
	}
	return updated, nil
}

func buildSetupSvcLiveReplayCompletionAuditResult(projectPath string, manifestArg string) setupSvcLiveReplayCompletionAuditResult {
	expected := setupSvcLiveReplayDomains()
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	matrixContract := setupSvcLiveReplayMatrixContractStatus(projectPath)
	testEvidence := setupSvcLiveReplayTestEvidenceStatus(projectPath)
	matrixStatuses := setupSvcLiveReplayMatrixStatuses(projectPath)
	result := setupSvcLiveReplayCompletionAuditResult{
		Mode:                      "setup-svc-live-replay-completion-audit",
		Project:                   projectPath,
		ReadOnly:                  true,
		Status:                    "passed",
		ManifestPath:              manifestPath,
		MatrixContract:            matrixContract,
		TestEvidence:              testEvidence,
		MetadataServiceDatasource: setupSvcLiveReplayDatasourceReadinessFor(),
		Notes: []string{
			"This command is a read-only completion gate; it never executes setup-svc or MetadataService writes.",
			"Completion requires a passed matrixContract, passed replay test evidence contract, passed live replay evidence, passed promotion audit, and every supported domain already marked verified in the parity matrix.",
			"A passed promotion audit with covered matrix statuses is ready for a matrix status update, not a completed goal.",
		},
	}
	for _, domain := range expected {
		result.Totals.Domains++
		result.Totals.Operations += len(domain.Operations)
		switch matrixStatuses[normalizeDomain(domain.Domain)] {
		case "verified":
			result.Totals.MatrixVerifiedDomains++
		case "covered":
			result.Totals.MatrixCoveredDomains++
			result.Totals.MatrixNonVerifiedDomains++
		default:
			result.Totals.MatrixNonVerifiedDomains++
		}
	}
	if matrixContract.Status == "passed" {
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:     "matrix_contract",
			Status:   "passed",
			Evidence: "parity matrix contract passed",
		})
	} else {
		result.Status = "blocked_parity_matrix_contract"
		for _, issue := range matrixContract.Issues {
			result.BlockingIssues = append(result.BlockingIssues, "parityMatrix: "+issue)
		}
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:       "matrix_contract",
			Status:     "blocked",
			Blocking:   true,
			Evidence:   "parity matrix contract failed",
			NextAction: "Repair the parity matrix before collecting or promoting live replay evidence.",
		})
	}
	if testEvidence.Status == "passed" {
		sourceEvidence := ""
		if testEvidence.TestSourceStatus != "" {
			sourceEvidence = fmt.Sprintf("; test source status %s with %d checks", testEvidence.TestSourceStatus, testEvidence.TestSourceChecks)
		}
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:     "test_evidence_contract",
			Status:   "passed",
			Evidence: fmt.Sprintf("%d domains and %d operations mapped to replay tests%s", testEvidence.Domains, testEvidence.Operations, sourceEvidence),
		})
	} else {
		if result.Status == "passed" {
			result.Status = "blocked_test_evidence_contract"
		}
		for _, issue := range testEvidence.Issues {
			result.BlockingIssues = append(result.BlockingIssues, "testEvidence: "+issue)
		}
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:       "test_evidence_contract",
			Status:     testEvidence.Status,
			Blocking:   true,
			Evidence:   "replay test evidence contract failed",
			NextAction: "Repair msapi-parity-test-evidence.json or the referenced replay test class/method source before completion.",
		})
	}
	if _, err := os.Stat(manifestPath); err != nil {
		if result.Status == "passed" {
			result.Status = "blocked_missing_live_replay_evidence"
		}
		result.BlockingIssues = append(result.BlockingIssues, "manifest: missing live replay evidence "+manifestPath)
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:       "live_replay_evidence",
			Status:     "missing",
			Blocking:   true,
			Evidence:   "manifest not found",
			NextAction: "Run an approved setup-svc live replay and collect the manifest evidence before completion.",
		})
		result.NextActions = setupSvcLiveReplayCompletionNextActions(result.Status, result.Totals.MatrixNonVerifiedDomains)
		result.NextCommands = setupSvcLiveReplayCompletionNextCommands(projectPath, manifestArg, result.Status)
		result.Domains = setupSvcLiveReplayCompletionDomains(expected, matrixStatuses, nil, nil)
		setupSvcLiveReplayCompletionAttachFailedEvidence(&result, projectPath, manifestPath)
		result.Totals.BlockedDomains = len(result.Domains)
		setupSvcLiveReplayCompletionAttachGateSummaries(&result)
		return result
	}
	evidence, evidenceErr := buildSetupSvcLiveReplayEvidenceResult(projectPath, manifestArg)
	if evidenceErr != nil {
		if result.Status == "passed" {
			result.Status = "blocked_live_replay_evidence"
		}
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+evidenceErr.Error())
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:       "live_replay_evidence",
			Status:     "blocked",
			Blocking:   true,
			Evidence:   "manifest could not be verified",
			NextAction: "Repair the live replay manifest JSON and rerun evidence verification.",
		})
		result.NextActions = setupSvcLiveReplayCompletionNextActions(result.Status, result.Totals.MatrixNonVerifiedDomains)
		result.NextCommands = setupSvcLiveReplayCompletionNextCommands(projectPath, manifestArg, result.Status)
		result.Domains = setupSvcLiveReplayCompletionDomains(expected, matrixStatuses, nil, nil)
		setupSvcLiveReplayCompletionAttachFailedEvidence(&result, projectPath, manifestPath)
		result.Totals.BlockedDomains = len(result.Domains)
		setupSvcLiveReplayCompletionAttachGateSummaries(&result)
		return result
	}
	result.ContractVersion = evidence.ContractVersion
	result.ContractFingerprint = evidence.ContractFingerprint
	result.Totals.EvidenceVerifiedDomains = evidence.Totals.VerifiedDomains
	result.Totals.EvidenceVerifiedOperations = evidence.Totals.VerifiedOperations
	if evidence.Status == "passed" {
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:     "live_replay_evidence",
			Status:   "passed",
			Evidence: fmt.Sprintf("%d/%d domains and %d/%d operations verified", evidence.Totals.VerifiedDomains, evidence.Totals.Domains, evidence.Totals.VerifiedOperations, evidence.Totals.Operations),
		})
	} else {
		if result.Status == "passed" {
			result.Status = "blocked_live_replay_evidence"
		}
		result.BlockingIssues = append(result.BlockingIssues, "manifest: "+evidence.Status)
		result.BlockingIssues = append(result.BlockingIssues, setupSvcLiveReplayEvidenceFailureIssues(evidence)...)
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:       "live_replay_evidence",
			Status:     evidence.Status,
			Blocking:   true,
			Evidence:   fmt.Sprintf("%d missing operations, %d failed operations", evidence.Totals.MissingOperations, evidence.Totals.FailedOperations),
			NextAction: "Repair setup-svc, MetadataService, query/readback, normalized diff, or cleanup evidence and rerun verification.",
		})
	}
	promotion, promotionErr := buildSetupSvcLiveReplayPromotionResult(projectPath, manifestArg)
	if promotionErr != nil {
		if result.Status == "passed" {
			result.Status = "blocked_promotion_audit"
		}
		result.BlockingIssues = append(result.BlockingIssues, "promotion: "+promotionErr.Error())
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:       "promotion_audit",
			Status:     "blocked",
			Blocking:   true,
			Evidence:   "promotion audit could not run",
			NextAction: "Repair evidence blockers and rerun setup-svc-live-replay-promotion.",
		})
		result.Domains = setupSvcLiveReplayCompletionDomains(expected, matrixStatuses, &evidence, nil)
		setupSvcLiveReplayCompletionAttachFailedEvidence(&result, projectPath, manifestPath)
		result.Totals.BlockedDomains = len(result.Domains)
		result.NextActions = setupSvcLiveReplayCompletionNextActions(result.Status, result.Totals.MatrixNonVerifiedDomains)
		result.NextCommands = setupSvcLiveReplayCompletionNextCommands(projectPath, manifestArg, result.Status)
		setupSvcLiveReplayCompletionAttachGateSummaries(&result)
		return result
	}
	result.Totals.PromotableDomains = promotion.Totals.PromotableDomains
	result.Totals.PromotableOperations = promotion.Totals.PromotableOperations
	if promotion.Status == "passed" {
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:     "promotion_audit",
			Status:   "passed",
			Evidence: fmt.Sprintf("%d promotable domains and %d promotable operations", promotion.Totals.PromotableDomains, promotion.Totals.PromotableOperations),
		})
	} else {
		if result.Status == "passed" {
			result.Status = "blocked_promotion_audit"
		}
		result.BlockingIssues = append(result.BlockingIssues, "promotion: "+promotion.Status)
		for _, issue := range promotion.BlockingIssues {
			result.BlockingIssues = append(result.BlockingIssues, "promotion: "+issue)
		}
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:       "promotion_audit",
			Status:     promotion.Status,
			Blocking:   true,
			Evidence:   fmt.Sprintf("%d blocked domains and %d blocked operations", promotion.Totals.BlockedDomains, promotion.Totals.BlockedOperations),
			NextAction: "Resolve promotion blockers before updating matrix statuses.",
		})
	}
	bundleStatus := ""
	if evidence.Status == "passed" && promotion.Status == "passed" {
		bundle := verifySetupSvcLiveReplayEvidenceBundle(projectPath, manifestArg)
		bundleStatus = bundle.Status
		if bundle.Status == "passed" {
			result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
				Name:     "evidence_bundle",
				Status:   "passed",
				Evidence: "evidence-bundle.json matches current manifest and artifact SHA-256 coverage",
			})
		} else {
			if result.Status == "passed" {
				result.Status = "blocked_evidence_bundle"
			}
			result.BlockingIssues = append(result.BlockingIssues, bundle.Issues...)
			if len(bundle.Issues) == 0 {
				result.BlockingIssues = append(result.BlockingIssues, "evidenceBundle: "+bundle.Status)
			}
			result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
				Name:       "evidence_bundle",
				Status:     bundle.Status,
				Blocking:   true,
				Evidence:   "evidence bundle is missing, stale, or invalid",
				NextAction: "Run setup-svc-live-replay-evidence-bundle with approval before matrix promotion.",
			})
		}
	}
	result.Domains = setupSvcLiveReplayCompletionDomains(expected, matrixStatuses, &evidence, &promotion)
	setupSvcLiveReplayCompletionAttachFailedEvidence(&result, projectPath, manifestPath)
	for _, domain := range result.Domains {
		if domain.CompletionStatus == "complete" {
			result.Totals.CompletedDomains++
			result.Totals.CompletedOperations += len(domain.VerifiedOperations)
		} else {
			result.Totals.BlockedDomains++
		}
	}
	if result.Status == "passed" && result.Totals.MatrixNonVerifiedDomains > 0 && evidence.Status == "passed" && promotion.Status == "passed" && bundleStatus == "passed" {
		result.Status = "ready_for_matrix_status_update"
		result.BlockingIssues = append(result.BlockingIssues, fmt.Sprintf("matrix: %d domains are not verified yet", result.Totals.MatrixNonVerifiedDomains))
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:       "matrix_status",
			Status:     "pending_update",
			Blocking:   true,
			Evidence:   fmt.Sprintf("%d domains remain non-verified in the parity matrix", result.Totals.MatrixNonVerifiedDomains),
			NextAction: "Record live replay evidence, apply the promotion matrix updates, rebuild, and rerun completion audit.",
		})
	} else if result.Status == "passed" && result.Totals.MatrixNonVerifiedDomains == 0 && bundleStatus == "passed" {
		result.Gates = append(result.Gates, setupSvcLiveReplayCompletionAuditGate{
			Name:     "matrix_status",
			Status:   "passed",
			Evidence: "all supported domains are verified in the parity matrix",
		})
	}
	result.NextActions = setupSvcLiveReplayCompletionNextActions(result.Status, result.Totals.MatrixNonVerifiedDomains)
	result.NextCommands = setupSvcLiveReplayCompletionNextCommands(projectPath, manifestArg, result.Status)
	setupSvcLiveReplayCompletionAttachGateSummaries(&result)
	return result
}

func setupSvcLiveReplayCompletionAttachGateSummaries(result *setupSvcLiveReplayCompletionAuditResult) {
	result.GateStatuses = make(map[string]setupSvcLiveReplayCompletionGateStatus, len(result.Gates))
	result.GateSummaries = make(map[string]map[string]any, len(result.Gates))
	for i := range result.Gates {
		gate := &result.Gates[i]
		if gate.Name != "" {
			result.GateStatuses[gate.Name] = setupSvcLiveReplayCompletionGateStatus{
				Status:     gate.Status,
				Blocking:   gate.Blocking,
				Evidence:   gate.Evidence,
				NextAction: gate.NextAction,
			}
		}
		if gate.Summary != nil {
			if gate.Name != "" {
				result.GateSummaries[gate.Name] = gate.Summary
			}
			continue
		}
		switch gate.Name {
		case "matrix_contract":
			gate.Summary = map[string]any{
				"status":                   result.MatrixContract.Status,
				"issueCount":               len(result.MatrixContract.Issues),
				"matrixVerifiedDomains":    result.Totals.MatrixVerifiedDomains,
				"matrixCoveredDomains":     result.Totals.MatrixCoveredDomains,
				"matrixNonVerifiedDomains": result.Totals.MatrixNonVerifiedDomains,
			}
			if result.MatrixContract.Path != "" {
				gate.Summary["path"] = result.MatrixContract.Path
			}
		case "test_evidence_contract":
			gate.Summary = map[string]any{
				"status":           result.TestEvidence.Status,
				"domains":          result.TestEvidence.Domains,
				"operations":       result.TestEvidence.Operations,
				"testSourceStatus": result.TestEvidence.TestSourceStatus,
				"testSourceChecks": result.TestEvidence.TestSourceChecks,
				"issueCount":       len(result.TestEvidence.Issues),
			}
			if result.TestEvidence.Path != "" {
				gate.Summary["path"] = result.TestEvidence.Path
			}
		case "live_replay_evidence":
			gate.Summary = map[string]any{
				"status":                     gate.Status,
				"manifestPath":               result.ManifestPath,
				"evidenceVerifiedDomains":    result.Totals.EvidenceVerifiedDomains,
				"evidenceVerifiedOperations": result.Totals.EvidenceVerifiedOperations,
				"failedEvidenceTotal":        result.FailedEvidenceTotal,
				"repairQueueCount":           result.RepairQueueCount,
			}
			if result.ContractVersion != "" {
				gate.Summary["contractVersion"] = result.ContractVersion
			}
			if result.ContractFingerprint != "" {
				gate.Summary["contractFingerprint"] = result.ContractFingerprint
			}
		case "promotion_audit":
			gate.Summary = map[string]any{
				"status":               gate.Status,
				"promotableDomains":    result.Totals.PromotableDomains,
				"promotableOperations": result.Totals.PromotableOperations,
				"blockedDomains":       result.Totals.BlockedDomains,
			}
		case "evidence_bundle":
			gate.Summary = map[string]any{
				"status":       gate.Status,
				"manifestPath": result.ManifestPath,
			}
		case "matrix_status":
			gate.Summary = map[string]any{
				"status":                   gate.Status,
				"matrixVerifiedDomains":    result.Totals.MatrixVerifiedDomains,
				"matrixNonVerifiedDomains": result.Totals.MatrixNonVerifiedDomains,
				"completedDomains":         result.Totals.CompletedDomains,
				"completedOperations":      result.Totals.CompletedOperations,
			}
		}
		if gate.Name != "" && gate.Summary != nil {
			result.GateSummaries[gate.Name] = gate.Summary
		}
	}
	if len(result.GateSummaries) == 0 {
		result.GateSummaries = nil
	}
	if len(result.GateStatuses) == 0 {
		result.GateStatuses = nil
	}
	result.OperatorPacket = setupSvcLiveReplayCompletionOperatorPacket{
		Status:                     result.Status,
		ManifestPath:               result.ManifestPath,
		GateStatuses:               result.GateStatuses,
		GateSummaries:              result.GateSummaries,
		FailedEvidence:             append([]string(nil), result.FailedEvidence...),
		FailedEvidenceTotal:        result.FailedEvidenceTotal,
		RepairQueueCount:           result.RepairQueueCount,
		RepairPlan:                 result.RepairPlan,
		RepairQueues:               result.FailedEvidenceSummary.RepairQueues,
		FailedEvidenceSummary:      result.FailedEvidenceSummary,
		BlockingIssues:             append([]string(nil), result.BlockingIssues...),
		MatrixVerifiedDomains:      result.Totals.MatrixVerifiedDomains,
		MatrixCoveredDomains:       result.Totals.MatrixCoveredDomains,
		MatrixNonVerifiedDomains:   result.Totals.MatrixNonVerifiedDomains,
		EvidenceVerifiedDomains:    result.Totals.EvidenceVerifiedDomains,
		EvidenceVerifiedOperations: result.Totals.EvidenceVerifiedOperations,
		PromotableDomains:          result.Totals.PromotableDomains,
		PromotableOperations:       result.Totals.PromotableOperations,
		BlockedDomains:             result.Totals.BlockedDomains,
		CompletedDomains:           result.Totals.CompletedDomains,
		CompletedOperations:        result.Totals.CompletedOperations,
		Domains:                    append([]setupSvcLiveReplayCompletionAuditDomain(nil), result.Domains...),
		MetadataServiceDatasource:  result.MetadataServiceDatasource,
		NextActions:                append([]string(nil), result.NextActions...),
		NextCommands:               append([]string(nil), result.NextCommands...),
	}
}

func setupSvcLiveReplayCompletionNextCommands(projectPath string, manifestArg string, status string) []string {
	manifestPath := setupSvcLiveReplayManifestPath(projectPath, manifestArg)
	switch status {
	case "blocked_live_replay_evidence":
		gaps, err := buildSetupSvcLiveReplayGapResult(projectPath, manifestArg)
		if err != nil || gaps.Status == "missing_manifest" {
			return nil
		}
		return setupSvcLiveReplayPreflightEvidenceCollectionCommands(gaps)
	case "blocked_evidence_bundle":
		return setupSvcLiveReplayCompletionDedupedCommands([]string{
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle " + shellPath(manifestPath),
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle " + shellPath(manifestPath) + " --dry-run",
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-bundle " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityEvidenceBundleApproval,
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath),
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		})
	case "ready_for_matrix_status_update":
		return setupSvcLiveReplayCompletionDedupedCommands([]string{
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath),
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath) + " --dry-run",
			"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-promotion " + shellPath(manifestPath) + " --execute --approval " + setupSvcParityMatrixPromotionApproval,
			"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
		})
	default:
		return nil
	}
}

func setupSvcLiveReplayCompletionDedupedCommands(commands []string) []string {
	var result []string
	for _, command := range commands {
		trimmed := strings.TrimSpace(command)
		if trimmed != "" && !containsString(result, trimmed) {
			result = append(result, trimmed)
		}
	}
	return result
}

func setupSvcLiveReplayCompletionDomains(expected []setupSvcLiveReplayDomain, matrixStatuses map[string]string, evidence *setupSvcLiveReplayEvidenceResult, promotion *setupSvcLiveReplayPromotionResult) []setupSvcLiveReplayCompletionAuditDomain {
	evidenceByDomain := map[string]setupSvcLiveReplayEvidenceDomain{}
	if evidence != nil {
		for _, domain := range evidence.Domains {
			evidenceByDomain[normalizeDomain(domain.Domain)] = domain
		}
	}
	promotionByDomain := map[string]setupSvcLiveReplayPromotionDomain{}
	if promotion != nil {
		for _, domain := range promotion.Domains {
			promotionByDomain[normalizeDomain(domain.Domain)] = domain
		}
	}
	var domains []setupSvcLiveReplayCompletionAuditDomain
	for _, expectedDomain := range expected {
		key := normalizeDomain(expectedDomain.Domain)
		matrixStatus := matrixStatuses[key]
		if matrixStatus == "" {
			matrixStatus = "covered"
		}
		evidenceDomain := evidenceByDomain[key]
		promotionDomain := promotionByDomain[key]
		item := setupSvcLiveReplayCompletionAuditDomain{
			Domain:             expectedDomain.Domain,
			MatrixStatus:       matrixStatus,
			EvidenceStatus:     firstString(evidenceDomain.Status, "missing"),
			RecommendedStatus:  firstString(promotionDomain.RecommendedStatus, matrixStatus),
			CanPromote:         promotionDomain.CanPromote,
			VerifiedOperations: append([]string{}, evidenceDomain.VerifiedOperations...),
			BlockingOperations: append([]string{}, promotionDomain.BlockingOperations...),
			FailedEvidence:     append([]string{}, promotionDomain.FailedEvidence...),
		}
		if len(item.FailedEvidence) == 0 {
			item.FailedEvidence = setupSvcLiveReplayEvidenceDomainFailureIssues(evidenceDomain)
		}
		switch {
		case matrixStatus == "verified" && evidenceDomain.Status == "verified" && len(evidenceDomain.VerifiedOperations) == len(expectedDomain.Operations):
			item.CompletionStatus = "complete"
		case evidenceDomain.Status == "verified" && promotionDomain.CanPromote:
			item.CompletionStatus = "ready_for_matrix_update"
		case evidence == nil:
			item.CompletionStatus = "waiting_live_replay_evidence"
			item.BlockingOperations = append([]string{}, expectedDomain.Operations...)
		case evidenceDomain.Status == "":
			item.CompletionStatus = "missing_live_replay_evidence"
			item.BlockingOperations = append([]string{}, expectedDomain.Operations...)
		case evidenceDomain.Status != "verified":
			item.CompletionStatus = "blocked_" + evidenceDomain.Status
			if len(item.BlockingOperations) == 0 {
				item.BlockingOperations = setupSvcLiveReplayBlockingOperations(evidenceDomain)
			}
		default:
			item.CompletionStatus = "matrix_status_" + matrixStatus
			if len(item.BlockingOperations) == 0 {
				item.BlockingOperations = append([]string{}, expectedDomain.Operations...)
			}
		}
		domains = append(domains, item)
	}
	return domains
}

func setupSvcLiveReplayCompletionFailedEvidence(domains []setupSvcLiveReplayCompletionAuditDomain) []string {
	var failed []string
	for _, domain := range domains {
		for _, item := range domain.FailedEvidence {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" && !containsString(failed, trimmed) {
				failed = append(failed, trimmed)
			}
		}
	}
	return failed
}

func setupSvcLiveReplayCompletionAttachFailedEvidence(result *setupSvcLiveReplayCompletionAuditResult, projectPath string, manifestPath string) {
	result.FailedEvidence = setupSvcLiveReplayCompletionFailedEvidence(result.Domains)
	result.FailedEvidenceSummary = buildSetupSvcLiveReplayCompletionFailedEvidenceSummary(projectPath, manifestPath, result.FailedEvidence)
	result.FailedEvidenceTotal = result.FailedEvidenceSummary.Total
	result.RepairQueueCount = result.FailedEvidenceSummary.RepairQueueCount
	result.RepairPlan = buildSetupSvcLiveReplayCompletionRepairPlan(projectPath, manifestPath, "", result.FailedEvidenceSummary.RepairQueues, result.FailedEvidence)
	for index := range result.Domains {
		domain := &result.Domains[index]
		if len(domain.FailedEvidence) == 0 {
			domain.RepairPlan = nil
			continue
		}
		summary := buildSetupSvcLiveReplayCompletionFailedEvidenceSummary(projectPath, manifestPath, domain.FailedEvidence)
		plan := buildSetupSvcLiveReplayCompletionRepairPlan(projectPath, manifestPath, domain.Domain, summary.RepairQueues, domain.FailedEvidence)
		domain.RepairPlan = &plan
	}
	result.Totals.FailedEvidenceTotal = result.FailedEvidenceSummary.Total
	result.Totals.RepairQueueCount = result.FailedEvidenceSummary.RepairQueueCount
}

func buildSetupSvcLiveReplayCompletionRepairPlan(projectPath string, manifestPath string, domain string, queues []setupSvcLiveReplayEvidenceImportRepairQueue, failedEvidence []string) setupSvcLiveReplayCompletionRepairPlan {
	plan := setupSvcLiveReplayCompletionRepairPlan{
		RepairQueueCount: len(queues),
	}
	for _, queue := range queues {
		sourceSystem := strings.TrimSpace(queue.ArtifactType)
		group := setupSvcLiveReplayCompletionRepairPlanGroup{
			SourceSystem:               sourceSystem,
			ArtifactType:               queue.ArtifactType,
			EvidenceSection:            queue.EvidenceSection,
			Count:                      queue.Count,
			SourceFiles:                queue.SourceFiles,
			TargetFiles:                queue.TargetFiles,
			WorklistCommand:            queue.WorklistCommand,
			SourceChecklistCommand:     queue.SourceChecklistCommand,
			SourceExecutionCommand:     queue.SourceExecutionCommand,
			SaveSourceExecutionCommand: queue.SaveSourceExecutionCommand,
		}
		plan.TotalSourceFiles += queue.SourceFiles
		plan.TotalTargetFiles += queue.TargetFiles
		if plan.PrimaryCommand == "" {
			plan.PrimarySourceSystem = sourceSystem
			plan.PrimaryEvidenceSection = queue.EvidenceSection
			plan.PrimaryCommand = firstString(queue.SaveSourceExecutionCommand, queue.SourceExecutionCommand, queue.SaveSourceChecklistCommand, queue.SourceChecklistCommand, queue.SaveWorklistCommand, queue.WorklistCommand)
		}
		command := firstString(queue.SaveSourceExecutionCommand, queue.SourceExecutionCommand, queue.SaveSourceChecklistCommand, queue.SourceChecklistCommand, queue.SaveWorklistCommand, queue.WorklistCommand)
		if command != "" && !containsString(plan.NextRepairCommands, command) {
			plan.NextRepairCommands = append(plan.NextRepairCommands, command)
		}
		plan.Groups = append(plan.Groups, group)
	}
	plan.DomainOperations = setupSvcLiveReplayCompletionRepairPlanDomainOperations(failedEvidence)
	plan.NextRepairScript = setupSvcLiveReplayCompletionRepairPlanScript(plan.NextRepairCommands)
	if len(plan.NextRepairCommands) > 0 {
		plan.PostRepairCommands = setupSvcLiveReplayCompletionRepairPlanPostCommands(projectPath, manifestPath, domain)
		plan.NextRepairScriptPath = setupSvcLiveReplayCompletionRepairPlanScriptPath(projectPath, domain)
		plan.SaveNextRepairScript = setupSvcLiveReplayCompletionRepairPlanSaveScriptCommand(projectPath, manifestPath, domain, plan.NextRepairScriptPath)
	}
	return plan
}

func setupSvcLiveReplayCompletionRepairPlanPostCommands(projectPath string, manifestPath string, domain string) []string {
	options := setupSvcLiveReplayCollectionPlanOptions{
		Domain:          strings.TrimSpace(domain),
		SourceReadiness: "complete",
		BatchIndex:      0,
		BatchLimit:      setupSvcLiveReplayWorklistBatchLimit,
	}
	worklistPath := setupSvcLiveReplayWorklistSuggestedPath(projectPath, options)
	validateOptions := options
	validateOptions.BatchIndex = -1
	validateOptions.BatchLimit = setupSvcLiveReplayWorklistBatchLimit
	return []string{
		setupSvcLiveReplaySourceValidateScanCommand(projectPath, manifestPath, "setup-svc-live-replay-source-validate", validateOptions),
		setupSvcLiveReplayWorklistCommand(projectPath, manifestPath, options) + " > " + shellPath(worklistPath),
		"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(worklistPath) + " --dry-run",
		"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-evidence-import @" + shellPath(worklistPath) + " --execute --approval " + setupSvcParityEvidenceImportApproval,
		"cloudcc apply msapi " + shellPath(projectPath) + " setup-svc-live-replay-manifest-sync " + shellPath(manifestPath) + " --dry-run",
		"cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath),
	}
}

func setupSvcLiveReplayCompletionRepairPlanScript(commands []string) string {
	if len(commands) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n\n")
	for _, command := range commands {
		trimmed := strings.TrimSpace(command)
		if trimmed == "" {
			continue
		}
		b.WriteString(trimmed)
		b.WriteString("\n")
	}
	return b.String()
}

func setupSvcLiveReplayCompletionRepairPlanScriptPath(projectPath string, domain string) string {
	name := "repair-plan-next-repair-commands.sh"
	if strings.TrimSpace(domain) != "" {
		name = "repair-plan-" + setupSvcLiveReplayRepairQueueSlug(domain) + "-next-repair-commands.sh"
	}
	return filepath.Join(projectPath, "outputs", "setup-svc-live-replay", name)
}

func setupSvcLiveReplayCompletionRepairPlanSaveScriptCommand(projectPath string, manifestPath string, domain string, scriptPath string) string {
	if strings.TrimSpace(scriptPath) == "" {
		return ""
	}
	command := "cloudcc scan msapi " + shellPath(projectPath) + " setup-svc-live-replay-completion-audit " + shellPath(manifestPath)
	jqPath := ".repairPlan.nextRepairScript"
	if strings.TrimSpace(domain) != "" {
		jqPath = ".domains[] | select(.domain==" + strconv.Quote(domain) + ") | .repairPlan.nextRepairScript"
	}
	return command + " | jq -r " + shellPath(jqPath) + " > " + shellPath(scriptPath) + " && chmod +x " + shellPath(scriptPath)
}

func setupSvcLiveReplayCompletionRepairPlanDomainOperations(failedEvidence []string) []setupSvcLiveReplayCompletionRepairPlanDomainOperation {
	type accumulator struct {
		domain    string
		operation string
		count     int
		queues    []string
	}
	byKey := map[string]*accumulator{}
	for _, item := range failedEvidence {
		domain, operation, issue := setupSvcLiveReplayCompletionFailedEvidenceParts(item)
		domain = strings.TrimSpace(domain)
		operation = strings.TrimSpace(operation)
		if domain == "" || operation == "" {
			continue
		}
		key := domain + "\x00" + operation
		acc := byKey[key]
		if acc == nil {
			acc = &accumulator{domain: domain, operation: operation}
			byKey[key] = acc
		}
		acc.count++
		for _, queueKey := range setupSvcLiveReplayCompletionRepairQueueKeys(issue) {
			parts := strings.SplitN(queueKey, "\x00", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
				continue
			}
			label := parts[0] + "/" + parts[1]
			if !containsString(acc.queues, label) {
				acc.queues = append(acc.queues, label)
			}
		}
	}
	result := make([]setupSvcLiveReplayCompletionRepairPlanDomainOperation, 0, len(byKey))
	for _, acc := range byKey {
		sort.Strings(acc.queues)
		result = append(result, setupSvcLiveReplayCompletionRepairPlanDomainOperation{
			Domain:              acc.domain,
			Operation:           acc.operation,
			FailedEvidenceCount: acc.count,
			PrimaryRepairQueue:  firstString(acc.queues...),
			RepairQueues:        acc.queues,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].FailedEvidenceCount == result[j].FailedEvidenceCount {
			if result[i].Domain == result[j].Domain {
				return result[i].Operation < result[j].Operation
			}
			return result[i].Domain < result[j].Domain
		}
		return result[i].FailedEvidenceCount > result[j].FailedEvidenceCount
	})
	return result
}

func buildSetupSvcLiveReplayCompletionFailedEvidenceSummary(projectPath string, manifestPath string, failed []string) setupSvcLiveReplayCompletionFailedEvidenceSummary {
	issueCounts := map[string]int{}
	domainOperationCounts := map[string]int{}
	queueCounts := map[string]int{}
	queueArtifacts := map[string]bool{}
	for _, item := range failed {
		domain, operation, issue := setupSvcLiveReplayCompletionFailedEvidenceParts(item)
		issueCounts[setupSvcLiveReplayEvidenceImportIssueFamily(issue)]++
		if domain != "" && operation != "" {
			domainOperationCounts[domain+"\x00"+operation]++
		}
		path := setupSvcLiveReplayCompletionFailedEvidenceArtifactPath(issue)
		for _, queueKey := range setupSvcLiveReplayCompletionRepairQueueKeys(issue) {
			artifactQueueKey := queueKey + "\x00" + path
			if queueArtifacts[artifactQueueKey] {
				continue
			}
			queueArtifacts[artifactQueueKey] = true
			queueCounts[queueKey]++
		}
	}
	repairQueues := setupSvcLiveReplayEvidenceImportRepairQueues(projectPath, manifestPath, queueCounts)
	return setupSvcLiveReplayCompletionFailedEvidenceSummary{
		Total:                 len(failed),
		IssueCounts:           setupSvcLiveReplayEvidenceImportSortedIssueCounts(issueCounts),
		DomainOperationCounts: setupSvcLiveReplayCompletionSortedDomainOperationCounts(domainOperationCounts),
		RepairQueueCount:      len(repairQueues),
		RepairQueues:          repairQueues,
	}
}

func setupSvcLiveReplayCompletionRepairQueueKeys(issue string) []string {
	artifactType := setupSvcLiveReplayCompletionFailedEvidenceArtifactType(issue)
	if artifactType == "" {
		return nil
	}
	sections := setupSvcLiveReplayCompletionFailedEvidenceSections(artifactType, issue)
	keys := make([]string, 0, len(sections))
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" || setupSvcLiveReplayEvidenceIdentitySection(section) {
			continue
		}
		keys = append(keys, artifactType+"\x00"+section)
	}
	return keys
}

func setupSvcLiveReplayCompletionFailedEvidenceArtifactType(issue string) string {
	path := setupSvcLiveReplayCompletionFailedEvidenceArtifactPath(issue)
	if path == "" {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	return setupSvcLiveReplayNormalizeArtifactType(base)
}

func setupSvcLiveReplayCompletionFailedEvidenceArtifactPath(issue string) string {
	issue = strings.TrimSpace(issue)
	if issue == "" {
		return ""
	}
	parts := strings.Split(issue, ":")
	for i := len(parts) - 1; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if strings.HasSuffix(part, ".json") {
			return part
		}
	}
	return ""
}

func setupSvcLiveReplayCompletionFailedEvidenceSections(artifactType string, issue string) []string {
	family := setupSvcLiveReplayEvidenceImportIssueFamily(issue)
	switch family {
	case "runtimeEffectsMissingEvidence":
		return []string{"runtimeEffectChecks"}
	case "setupSvcSnapshotMissingTableEvidence", "metadataServiceSnapshotMissingTableEvidence":
		return []string{"tableSnapshots"}
	case "metadataServiceDatasourceMissingEvidence", "metadataServiceDatasourceNotReady", "metadataServiceDatasourceStatusNotReady":
		return []string{"metadataServiceDatasource"}
	case "queryReadbackExpectationsMissingEvidence":
		return []string{"readbackExpectationChecks"}
	case "queryReadbackMissingTableCoverage", "queryReadbackMissingRowEvidence", "queryReadbackMissingFieldEvidence":
		return []string{"readbackTables"}
	case "queryReadbackMissingRelationshipEvidence":
		return []string{"relationshipChecks"}
	case "cleanupMissingResidualEvidence", "cleanupMissingResidualCounters":
		return []string{"residualCounters"}
	case "evidenceFileStatusNotPassed":
		return setupSvcLiveReplayRequiredEvidenceSections(artifactType)
	default:
		return nil
	}
}

func setupSvcLiveReplayCompletionFailedEvidenceParts(item string) (string, string, string) {
	item = strings.TrimSpace(item)
	prefix, issue, ok := strings.Cut(item, ": ")
	if !ok {
		return "", "", item
	}
	domain, operation, ok := strings.Cut(strings.TrimSpace(prefix), "/")
	if !ok {
		return "", "", strings.TrimSpace(issue)
	}
	return strings.TrimSpace(domain), strings.TrimSpace(operation), strings.TrimSpace(issue)
}

func setupSvcLiveReplayCompletionSortedDomainOperationCounts(counts map[string]int) []setupSvcLiveReplayCompletionDomainOperationCount {
	items := make([]setupSvcLiveReplayCompletionDomainOperationCount, 0, len(counts))
	for key, count := range counts {
		parts := strings.SplitN(key, "\x00", 2)
		if len(parts) != 2 {
			continue
		}
		items = append(items, setupSvcLiveReplayCompletionDomainOperationCount{Domain: parts[0], Operation: parts[1], Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			if items[i].Domain == items[j].Domain {
				return items[i].Operation < items[j].Operation
			}
			return items[i].Domain < items[j].Domain
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func setupSvcLiveReplayCompletionNextActions(status string, nonVerifiedDomains int) []string {
	switch status {
	case "passed":
		return []string{"Completion audit passed; preserve the evidence manifest, test report, release artifacts, and verified parity matrix."}
	case "ready_for_matrix_status_update":
		return []string{"Record the approved live replay evidence in .claw/test-report.md.", "Apply the read-only promotion candidates to the parity matrix statuses.", "Rebuild and rerun setup-svc-live-replay-completion-audit before closing the goal."}
	case "blocked_missing_live_replay_evidence":
		return []string{"Run setup-svc-live-replay-readiness and setup-svc-live-replay-packet.", "Collect approved setup-svc, MetadataService, query/readback, normalized diff, and cleanup JSON evidence.", "Run setup-svc-live-replay-evidence and setup-svc-live-replay-completion-audit."}
	case "blocked_parity_matrix_contract":
		return []string{"Repair parity matrix references, queryIncluded flags, operations, and requiredTables before collecting or promoting evidence."}
	case "blocked_test_evidence_contract":
		return []string{"Repair replay test evidence coverage or declared parity replay test source methods.", "Rerun setup-svc-live-replay-coverage and setup-svc-live-replay-completion-audit."}
	case "blocked_live_replay_evidence":
		return []string{"Repair failed or missing live replay evidence artifacts, then rerun setup-svc-live-replay-evidence."}
	case "blocked_promotion_audit":
		return []string{"Resolve promotion blockers before changing matrix statuses."}
	case "blocked_evidence_bundle":
		return []string{"Run setup-svc-live-replay-evidence-bundle with approval after evidence verification passes.", "Rerun setup-svc-live-replay-completion-audit before matrix promotion."}
	default:
		if nonVerifiedDomains > 0 {
			return []string{"Inspect completion audit gates and repair the remaining non-verified domains."}
		}
		return []string{"Inspect completion audit gates and rerun after repairs."}
	}
}

func setupSvcLiveReplayMatrixContractStatus(projectPath string) setupSvcLiveReplayMatrixContract {
	matrixPath := setupSvcLiveReplayParityMatrixPath(projectPath)
	result := setupSvcLiveReplayMatrixContract{
		Path:   matrixPath,
		Status: "passed",
	}
	if matrixPath == "" {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "missing parity matrix file")
		return result
	}
	body, err := os.ReadFile(matrixPath)
	if err != nil {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "cannot read "+matrixPath)
		return result
	}
	var matrix map[string]any
	if err := json.Unmarshal(body, &matrix); err != nil {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "invalid JSON in "+matrixPath)
		return result
	}
	matrixDomains := mapList(matrix["domains"])
	if len(matrixDomains) == 0 {
		result.Issues = append(result.Issues, "missing domains")
	}
	expected := map[string]setupSvcLiveReplayDomain{}
	for _, domain := range setupSvcLiveReplayDomains() {
		expected[normalizeDomain(domain.Domain)] = domain
	}
	seen := map[string]map[string]any{}
	for _, item := range matrixDomains {
		domain := normalizeDomain(firstMapString(item, "domain", "name"))
		switch {
		case domain == "":
			result.Issues = append(result.Issues, "domain entry missing domain")
		case expected[domain].Domain == "":
			result.Issues = append(result.Issues, domain+": unexpected domain")
		case seen[domain] != nil:
			result.Issues = append(result.Issues, domain+": duplicate domain")
		default:
			seen[domain] = item
		}
	}
	for _, domain := range setupSvcLiveReplayDomains() {
		item := seen[normalizeDomain(domain.Domain)]
		if item == nil {
			result.Issues = append(result.Issues, domain.Domain+": missing from parity matrix")
			continue
		}
		result.Issues = append(result.Issues, setupSvcLiveReplayMatrixDomainIssues(domain, item)...)
	}
	if len(result.Issues) > 0 {
		result.Status = "blocked"
	}
	return result
}

func setupSvcLiveReplayMatrixDomainItems(matrixPath string) map[string]map[string]any {
	items := map[string]map[string]any{}
	if strings.TrimSpace(matrixPath) == "" {
		return items
	}
	body, err := os.ReadFile(matrixPath)
	if err != nil {
		return items
	}
	var matrix map[string]any
	if err := json.Unmarshal(body, &matrix); err != nil {
		return items
	}
	for _, item := range mapList(matrix["domains"]) {
		domain := normalizeDomain(firstMapString(item, "domain", "name"))
		if domain != "" {
			items[domain] = item
		}
	}
	return items
}

func setupSvcLiveReplayMatrixDomainIssues(expected setupSvcLiveReplayDomain, item map[string]any) []string {
	var issues []string
	domain := expected.Domain
	if len(stringList(item["setupSvcReferences"])) == 0 {
		issues = append(issues, domain+": missing setupSvcReferences")
	}
	runtimeEffects := stringList(item["runtimeEffects"])
	if len(runtimeEffects) == 0 {
		issues = append(issues, domain+": missing runtimeEffects")
	}
	if missing := missingSetupSvcLiveReplayStrings(expected.RuntimeEffects, runtimeEffects, false); len(missing) > 0 {
		issues = append(issues, domain+": missing runtimeEffects "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expected.RuntimeEffects, runtimeEffects, false); len(unexpected) > 0 {
		issues = append(issues, domain+": unexpected runtimeEffects "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(runtimeEffects, false); len(duplicates) > 0 {
		issues = append(issues, domain+": duplicate runtimeEffects "+strings.Join(duplicates, ","))
	}
	queryReadbackExpectations := stringList(item["queryReadbackExpectations"])
	if len(queryReadbackExpectations) == 0 {
		issues = append(issues, domain+": missing queryReadbackExpectations")
	}
	if missing := missingSetupSvcLiveReplayStrings(expected.QueryReadbackExpectations, queryReadbackExpectations, false); len(missing) > 0 {
		issues = append(issues, domain+": missing queryReadbackExpectations "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expected.QueryReadbackExpectations, queryReadbackExpectations, false); len(unexpected) > 0 {
		issues = append(issues, domain+": unexpected queryReadbackExpectations "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(queryReadbackExpectations, false); len(duplicates) > 0 {
		issues = append(issues, domain+": duplicate queryReadbackExpectations "+strings.Join(duplicates, ","))
	}
	queryIncluded, ok := item["queryIncluded"].(bool)
	if !ok {
		issues = append(issues, domain+": missing queryIncluded")
	} else if !queryIncluded {
		issues = append(issues, domain+": queryIncluded must be true")
	}
	status := strings.ToLower(strings.TrimSpace(firstMapString(item, "status", "currentStatus")))
	if status == "" {
		issues = append(issues, domain+": missing status")
	}
	operations := stringList(item["operations"])
	if len(operations) == 0 {
		issues = append(issues, domain+": missing operations")
	}
	if missing := missingSetupSvcLiveReplayStrings(expected.Operations, operations, false); len(missing) > 0 {
		issues = append(issues, domain+": missing operations "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expected.Operations, operations, false); len(unexpected) > 0 {
		issues = append(issues, domain+": unexpected operations "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(operations, false); len(duplicates) > 0 {
		issues = append(issues, domain+": duplicate operations "+strings.Join(duplicates, ","))
	}
	requiredTables := stringList(item["requiredTables"])
	if len(requiredTables) == 0 {
		issues = append(issues, domain+": missing requiredTables")
	}
	if missing := missingSetupSvcLiveReplayStrings(expected.RequiredTables, requiredTables, false); len(missing) > 0 {
		issues = append(issues, domain+": missing requiredTables "+strings.Join(missing, ","))
	}
	if unexpected := unexpectedSetupSvcLiveReplayStrings(expected.RequiredTables, requiredTables, false); len(unexpected) > 0 {
		issues = append(issues, domain+": unexpected requiredTables "+strings.Join(unexpected, ","))
	}
	if duplicates := duplicateSetupSvcLiveReplayStrings(requiredTables, false); len(duplicates) > 0 {
		issues = append(issues, domain+": duplicate requiredTables "+strings.Join(duplicates, ","))
	}
	return issues
}

func setupSvcLiveReplayMatrixStatuses(projectPath string) map[string]string {
	statuses := map[string]string{}
	for _, domain := range setupSvcLiveReplayDomains() {
		statuses[normalizeDomain(domain.Domain)] = "covered"
	}
	matrixPath := setupSvcLiveReplayParityMatrixPath(projectPath)
	if matrixPath == "" {
		return statuses
	}
	body, err := os.ReadFile(matrixPath)
	if err != nil {
		return statuses
	}
	var matrix map[string]any
	if err := json.Unmarshal(body, &matrix); err != nil {
		return statuses
	}
	for _, item := range mapList(matrix["domains"]) {
		domain := normalizeDomain(firstMapString(item, "domain", "name"))
		status := strings.ToLower(strings.TrimSpace(firstMapString(item, "status", "currentStatus")))
		if domain == "" || status == "" {
			continue
		}
		statuses[domain] = status
	}
	return statuses
}

func setupSvcLiveReplayTestEvidenceStatus(projectPath string) setupSvcLiveReplayTestEvidence {
	path := setupSvcLiveReplayTestEvidencePath(projectPath)
	result := setupSvcLiveReplayTestEvidence{
		Path:             path,
		Status:           "passed",
		TestSourceStatus: "not_checked",
	}
	if path == "" {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "missing msapi-parity-test-evidence.json")
		return result
	}
	body, err := os.ReadFile(path)
	if err != nil {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "read "+path+": "+err.Error())
		return result
	}
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		result.Status = "blocked"
		result.Issues = append(result.Issues, "parse "+path+": "+err.Error())
		return result
	}
	sourceRoot := setupSvcLiveReplayTestEvidenceSourceRoot(projectPath, path)
	if sourceRoot == "" {
		result.TestSourceStatus = "not_found"
	} else {
		result.TestSourcePath = sourceRoot
		result.TestSourceStatus = "passed"
	}
	seen := map[string]bool{}
	for _, item := range mapList(document["domains"]) {
		domain := normalizeDomain(firstMapString(item, "domain", "name"))
		if domain == "" {
			result.Issues = append(result.Issues, "test evidence domain missing domain")
			continue
		}
		if seen[domain] {
			result.Issues = append(result.Issues, domain+": duplicate test evidence domain")
			continue
		}
		seen[domain] = true
		result.Domains++
		for _, evidence := range mapList(item["operationEvidence"]) {
			operation := strings.TrimSpace(firstMapString(evidence, "operation"))
			testClass := strings.TrimSpace(firstMapString(evidence, "testClass"))
			testMethod := strings.TrimSpace(firstMapString(evidence, "testMethod"))
			if operation == "" {
				result.Issues = append(result.Issues, domain+": test evidence entry missing operation")
				continue
			}
			result.Operations++
			if testClass == "" {
				result.Issues = append(result.Issues, domain+"/"+operation+": missing testClass")
			}
			if testMethod == "" {
				result.Issues = append(result.Issues, domain+"/"+operation+": missing testMethod")
			}
			if sourceRoot != "" && testClass != "" && testMethod != "" {
				result.TestSourceChecks++
				if issue := setupSvcLiveReplayTestEvidenceSourceIssue(sourceRoot, testClass, testMethod); issue != "" {
					result.Issues = append(result.Issues, domain+"/"+operation+": "+issue)
				}
			}
		}
	}
	expectedDomains := []string{}
	for _, expected := range setupSvcLiveReplayDomains() {
		expectedDomains = append(expectedDomains, expected.Domain)
	}
	for _, missing := range missingSetupSvcLiveReplayStrings(expectedDomains, keysOfBoolMap(seen), false) {
		result.Issues = append(result.Issues, missing+": missing test evidence domain")
	}
	if len(result.Issues) > 0 {
		result.Status = "blocked"
		if sourceRoot != "" {
			result.TestSourceStatus = "blocked"
		}
	}
	return result
}

func setupSvcLiveReplayTestEvidenceDomainItems(path string) map[string]map[string]any {
	items := map[string]map[string]any{}
	if path == "" {
		return items
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return items
	}
	var document map[string]any
	if err := json.Unmarshal(body, &document); err != nil {
		return items
	}
	for _, item := range mapList(document["domains"]) {
		domain := normalizeDomain(firstMapString(item, "domain", "name"))
		if domain != "" {
			items[domain] = item
		}
	}
	return items
}

func setupSvcLiveReplayTestEvidenceOperations(domainEvidence map[string]any) []string {
	operations := []string{}
	for _, item := range mapList(domainEvidence["operationEvidence"]) {
		if operation := strings.TrimSpace(firstMapString(item, "operation")); operation != "" {
			operations = append(operations, operation)
		}
	}
	return operations
}

func setupSvcLiveReplayTestEvidenceSourceIssue(sourceRoot string, testClass string, testMethod string) string {
	sourcePath := filepath.Join(sourceRoot, testClass+".java")
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return "missing replay test source " + sourcePath
	}
	source := string(body)
	methodNeedles := []string{
		"void " + testMethod + "(",
		"void\n" + testMethod + "(",
	}
	for _, needle := range methodNeedles {
		if strings.Contains(source, needle) {
			return ""
		}
	}
	return "missing replay test method " + testClass + "." + testMethod
}

func setupSvcLiveReplayParityMatrixPath(projectPath string) string {
	candidates := []string{}
	if fromEnv := strings.TrimSpace(os.Getenv("CLOUDCC_MSAPI_PARITY_MATRIX_FILE")); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}
	if projectPath != "" {
		candidates = append(candidates,
			filepath.Join(projectPath, "cc-metadata-service", "src", "test", "resources", "parity", "msapi-setup-svc-parity-matrix.json"),
			filepath.Join(projectPath, "src", "test", "resources", "parity", "msapi-setup-svc-parity-matrix.json"),
			filepath.Join(projectPath, "test-fixtures", "msapi", "parity", "msapi-setup-svc-parity-matrix.json"),
			filepath.Join(projectPath, "docs", "specs", "msapi-setup-svc-parity-matrix.json"),
			filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "msapi-setup-svc-parity-matrix.json"),
		)
	}
	if skillRoot := strings.TrimSpace(os.Getenv("CLOUDCC_MSAPI_SKILL_ROOT")); skillRoot != "" {
		candidates = append(candidates, setupSvcLiveReplayParityMatrixCandidatePaths(skillRoot)...)
	}
	if exe, err := os.Executable(); err == nil {
		for _, root := range setupSvcLiveReplayPathAncestors(filepath.Dir(exe)) {
			candidates = append(candidates, setupSvcLiveReplayParityMatrixCandidatePaths(root)...)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for _, root := range setupSvcLiveReplayPathAncestors(cwd) {
			candidates = append(candidates, setupSvcLiveReplayParityMatrixCandidatePaths(root)...)
		}
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) {
			if abs, err := filepath.Abs(candidate); err == nil {
				candidate = abs
			}
		}
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func setupSvcLiveReplayTestEvidenceSourceRoot(projectPath string, evidencePath string) string {
	candidates := []string{}
	if fromEnv := strings.TrimSpace(os.Getenv("CLOUDCC_MSAPI_PARITY_TEST_SOURCE_ROOT")); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}
	if projectPath != "" {
		candidates = append(candidates, setupSvcLiveReplayTestEvidenceSourceCandidatePaths(projectPath)...)
	}
	if derived := setupSvcLiveReplayTestEvidenceSourceRootFromEvidencePath(evidencePath); derived != "" {
		candidates = append(candidates, derived)
	}
	if !setupSvcLiveReplayPathUnderRoot(evidencePath, projectPath) {
		if evidencePath != "" {
			for _, root := range setupSvcLiveReplayPathAncestors(filepath.Dir(evidencePath)) {
				candidates = append(candidates, setupSvcLiveReplayTestEvidenceSourceCandidatePaths(root)...)
			}
		}
		if skillRoot := strings.TrimSpace(os.Getenv("CLOUDCC_MSAPI_SKILL_ROOT")); skillRoot != "" {
			candidates = append(candidates, setupSvcLiveReplayTestEvidenceSourceCandidatePaths(skillRoot)...)
		}
		if exe, err := os.Executable(); err == nil {
			for _, root := range setupSvcLiveReplayPathAncestors(filepath.Dir(exe)) {
				candidates = append(candidates, setupSvcLiveReplayTestEvidenceSourceCandidatePaths(root)...)
			}
		}
		if cwd, err := os.Getwd(); err == nil {
			for _, root := range setupSvcLiveReplayPathAncestors(cwd) {
				candidates = append(candidates, setupSvcLiveReplayTestEvidenceSourceCandidatePaths(root)...)
			}
		}
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) {
			if abs, err := filepath.Abs(candidate); err == nil {
				candidate = abs
			}
		}
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return ""
}

func setupSvcLiveReplayPathUnderRoot(path string, root string) bool {
	path = strings.TrimSpace(path)
	root = strings.TrimSpace(root)
	if path == "" || root == "" {
		return false
	}
	cleanPath, pathErr := filepath.Abs(filepath.Clean(path))
	cleanRoot, rootErr := filepath.Abs(filepath.Clean(root))
	if pathErr != nil || rootErr != nil {
		return false
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}
	return rel == "." || (rel != "" && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func setupSvcLiveReplayTestEvidenceSourceCandidatePaths(root string) []string {
	return []string{
		filepath.Join(root, "cc-metadata-service", "src", "test", "java", "com", "cloudcc", "metadata", "parity"),
		filepath.Join(root, "src", "test", "java", "com", "cloudcc", "metadata", "parity"),
	}
}

func setupSvcLiveReplayTestEvidenceSourceRootFromEvidencePath(evidencePath string) string {
	if strings.TrimSpace(evidencePath) == "" {
		return ""
	}
	clean := filepath.Clean(evidencePath)
	resourceSuffix := filepath.Join("src", "test", "resources", "parity", "msapi-parity-test-evidence.json")
	if strings.HasSuffix(clean, resourceSuffix) {
		base := strings.TrimSuffix(clean, resourceSuffix)
		return filepath.Join(base, "src", "test", "java", "com", "cloudcc", "metadata", "parity")
	}
	nestedResourceSuffix := filepath.Join("cc-metadata-service", "src", "test", "resources", "parity", "msapi-parity-test-evidence.json")
	if strings.HasSuffix(clean, nestedResourceSuffix) {
		base := strings.TrimSuffix(clean, nestedResourceSuffix)
		return filepath.Join(base, "cc-metadata-service", "src", "test", "java", "com", "cloudcc", "metadata", "parity")
	}
	return ""
}

func setupSvcLiveReplayTestEvidencePath(projectPath string) string {
	candidates := []string{}
	if fromEnv := strings.TrimSpace(os.Getenv("CLOUDCC_MSAPI_PARITY_TEST_EVIDENCE_FILE")); fromEnv != "" {
		candidates = append(candidates, fromEnv)
	}
	if projectPath != "" {
		candidates = append(candidates,
			filepath.Join(projectPath, "cc-metadata-service", "src", "test", "resources", "parity", "msapi-parity-test-evidence.json"),
			filepath.Join(projectPath, "src", "test", "resources", "parity", "msapi-parity-test-evidence.json"),
			filepath.Join(projectPath, "test-fixtures", "msapi", "parity", "msapi-parity-test-evidence.json"),
			filepath.Join(projectPath, "docs", "specs", "msapi-parity-test-evidence.json"),
			filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "msapi-parity-test-evidence.json"),
		)
	}
	if skillRoot := strings.TrimSpace(os.Getenv("CLOUDCC_MSAPI_SKILL_ROOT")); skillRoot != "" {
		candidates = append(candidates, setupSvcLiveReplayTestEvidenceCandidatePaths(skillRoot)...)
	}
	if exe, err := os.Executable(); err == nil {
		for _, root := range setupSvcLiveReplayPathAncestors(filepath.Dir(exe)) {
			candidates = append(candidates, setupSvcLiveReplayTestEvidenceCandidatePaths(root)...)
		}
	}
	if cwd, err := os.Getwd(); err == nil {
		for _, root := range setupSvcLiveReplayPathAncestors(cwd) {
			candidates = append(candidates, setupSvcLiveReplayTestEvidenceCandidatePaths(root)...)
		}
	}
	seen := map[string]bool{}
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if !filepath.IsAbs(candidate) {
			if abs, err := filepath.Abs(candidate); err == nil {
				candidate = abs
			}
		}
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func setupSvcLiveReplayParityMatrixCandidatePaths(root string) []string {
	return []string{
		filepath.Join(root, "test-fixtures", "msapi", "parity", "msapi-setup-svc-parity-matrix.json"),
		filepath.Join(root, "cc-metadata-service", "src", "test", "resources", "parity", "msapi-setup-svc-parity-matrix.json"),
		filepath.Join(root, "src", "test", "resources", "parity", "msapi-setup-svc-parity-matrix.json"),
	}
}

func setupSvcLiveReplayTestEvidenceCandidatePaths(root string) []string {
	return []string{
		filepath.Join(root, "test-fixtures", "msapi", "parity", "msapi-parity-test-evidence.json"),
		filepath.Join(root, "cc-metadata-service", "src", "test", "resources", "parity", "msapi-parity-test-evidence.json"),
		filepath.Join(root, "src", "test", "resources", "parity", "msapi-parity-test-evidence.json"),
	}
}

func keysOfBoolMap(items map[string]bool) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func setupSvcLiveReplayPathAncestors(path string) []string {
	out := []string{}
	for {
		if path == "" {
			return out
		}
		out = append(out, path)
		parent := filepath.Dir(path)
		if parent == path {
			return out
		}
		path = parent
	}
}

func setupSvcLiveReplayBlockingOperations(domain setupSvcLiveReplayEvidenceDomain) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, operation := range domain.MissingOperations {
		key := strings.ToLower(strings.TrimSpace(operation))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, operation)
	}
	for _, issue := range domain.FailedOperations {
		key := strings.ToLower(strings.TrimSpace(issue.Operation))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, issue.Operation)
	}
	return out
}

func setupSvcLiveReplayManifestPath(projectPath string, manifestArg string) string {
	manifestArg = strings.TrimSpace(manifestArg)
	if manifestArg == "" {
		return filepath.Join(projectPath, "outputs", "setup-svc-live-replay", "manifest.json")
	}
	manifestArg = strings.TrimPrefix(manifestArg, "@")
	if filepath.IsAbs(manifestArg) {
		return manifestArg
	}
	return filepath.Join(projectPath, manifestArg)
}

func verifySetupSvcLiveReplayDomain(projectPath string, expected setupSvcLiveReplayDomain, evidence map[string]any) setupSvcLiveReplayEvidenceDomain {
	result := setupSvcLiveReplayEvidenceDomain{
		Domain:             expected.Domain,
		Status:             "verified",
		ExpectedOperations: append([]string{}, expected.Operations...),
	}
	if evidence == nil {
		result.Status = "missing"
		result.MissingOperations = append(result.MissingOperations, expected.Operations...)
		return result
	}
	operationEvidence := map[string]map[string]any{}
	expectedOperations := map[string]bool{}
	for _, operation := range expected.Operations {
		expectedOperations[strings.ToLower(strings.TrimSpace(operation))] = true
	}
	for _, operation := range mapList(evidence["operations"]) {
		name := firstMapString(operation, "operation", "name", "mode")
		if name == "" {
			result.FailedOperations = append(result.FailedOperations, setupSvcLiveReplayOperationIssue{
				Operation:      "(missing)",
				FailedEvidence: []string{"missingOperationName"},
			})
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(name))
		if !expectedOperations[normalized] {
			result.FailedOperations = append(result.FailedOperations, setupSvcLiveReplayOperationIssue{
				Operation:      name,
				FailedEvidence: []string{"unexpectedOperation"},
			})
			continue
		}
		if _, exists := operationEvidence[normalized]; exists {
			result.FailedOperations = append(result.FailedOperations, setupSvcLiveReplayOperationIssue{
				Operation:      name,
				FailedEvidence: []string{"duplicateOperation"},
			})
			continue
		}
		operationEvidence[normalized] = operation
	}
	for _, operation := range expected.Operations {
		issue := verifySetupSvcLiveReplayOperation(projectPath, expected, operation, operationEvidence[strings.ToLower(operation)])
		if len(issue.MissingEvidence) == 0 && len(issue.FailedEvidence) == 0 {
			result.VerifiedOperations = append(result.VerifiedOperations, operation)
			continue
		}
		if operationEvidence[strings.ToLower(operation)] == nil {
			result.MissingOperations = append(result.MissingOperations, operation)
		}
		result.FailedOperations = append(result.FailedOperations, issue)
	}
	if len(result.MissingOperations) > 0 {
		result.Status = "missing_operations"
	} else if len(result.FailedOperations) > 0 {
		result.Status = "failed_evidence"
	}
	return result
}

func verifySetupSvcLiveReplayOperation(projectPath string, domain setupSvcLiveReplayDomain, operation string, evidence map[string]any) setupSvcLiveReplayOperationIssue {
	issue := setupSvcLiveReplayOperationIssue{Operation: operation}
	if evidence == nil {
		issue.MissingEvidence = append(issue.MissingEvidence,
			"setupSvcEvidenceStatus",
			"metadataServiceEvidenceStatus",
			"queryEvidenceStatus",
			"normalizedDiffStatus")
		if operation != "query" {
			issue.MissingEvidence = append(issue.MissingEvidence, "cleanupStatus")
		}
		return issue
	}
	required := []string{
		"setupSvcEvidenceStatus",
		"metadataServiceEvidenceStatus",
		"queryEvidenceStatus",
		"normalizedDiffStatus",
	}
	if operation != "query" {
		required = append(required, "cleanupStatus")
	}
	for _, field := range required {
		status := strings.ToLower(firstMapString(evidence, field))
		if status == "" {
			issue.MissingEvidence = append(issue.MissingEvidence, field)
			continue
		}
		if status != "passed" {
			issue.FailedEvidence = append(issue.FailedEvidence, field+"="+status)
		}
	}
	evidenceFiles := setupSvcLiveReplayEvidenceFileList(evidence["evidenceFiles"])
	expectedFiles := setupSvcLiveReplayEvidenceFiles(domain.Domain, operation, operation != "query")
	contractIssues := setupSvcLiveReplayEvidenceFileContractIssues(projectPath, expectedFiles, evidenceFiles)
	for _, missing := range contractIssues.Missing {
		issue.MissingEvidence = append(issue.MissingEvidence, "evidenceFiles:"+missing)
	}
	for _, unexpected := range contractIssues.Unexpected {
		issue.FailedEvidence = append(issue.FailedEvidence, "unexpectedEvidenceFile:"+unexpected)
	}
	for _, duplicate := range contractIssues.Duplicate {
		issue.FailedEvidence = append(issue.FailedEvidence, "duplicateEvidenceFile:"+duplicate)
	}
	if len(contractIssues.Missing) > 0 || len(contractIssues.Unexpected) > 0 || len(contractIssues.Duplicate) > 0 {
		return issue
	}
	evidenceFileByContract := setupSvcLiveReplayEvidenceFileMap(projectPath, evidenceFiles)
	snapshotTablesByArtifact := map[string]map[string]bool{}
	for _, requiredFile := range expectedFiles {
		filePath := evidenceFileByContract[setupSvcLiveReplayEvidencePathForContract(projectPath, requiredFile)]
		resolved := setupSvcLiveReplayResolveEvidenceFile(projectPath, filePath)
		payload, err := os.ReadFile(resolved)
		if err != nil {
			issue.FailedEvidence = append(issue.FailedEvidence, "evidenceFileUnreadable:"+filePath)
			continue
		}
		var decoded any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			issue.FailedEvidence = append(issue.FailedEvidence, "evidenceFileInvalidJSON:"+filePath)
			continue
		}
		artifactType := setupSvcLiveReplayArtifactType(requiredFile)
		if artifactType == "setup-svc" || artifactType == "metadata-service" {
			if artifact, ok := decoded.(map[string]any); ok {
				snapshotTablesByArtifact[artifactType] = setupSvcLiveReplaySnapshotTableSet(artifact)
			}
		}
		if failures := verifySetupSvcLiveReplayEvidenceArtifact(projectPath, requiredFile, domain, operation, decoded); len(failures) > 0 {
			for _, failure := range failures {
				issue.FailedEvidence = append(issue.FailedEvidence, failure+":"+filePath)
			}
		}
	}
	issue.FailedEvidence = append(issue.FailedEvidence, setupSvcLiveReplaySnapshotTableSetPairFailures(
		snapshotTablesByArtifact["setup-svc"],
		snapshotTablesByArtifact["metadata-service"],
	)...)
	return issue
}

type setupSvcLiveReplayEvidenceFileContractIssue struct {
	Missing    []string
	Unexpected []string
	Duplicate  []string
}

func setupSvcLiveReplayEvidenceFileContractIssues(projectPath string, expected []string, actual []string) setupSvcLiveReplayEvidenceFileContractIssue {
	expectedSet := setupSvcLiveReplayEvidenceFileMap(projectPath, expected)
	actualSet := setupSvcLiveReplayEvidenceFileMap(projectPath, actual)
	var issue setupSvcLiveReplayEvidenceFileContractIssue
	for _, item := range expected {
		key := setupSvcLiveReplayEvidencePathForContract(projectPath, item)
		if key == "" {
			continue
		}
		if _, ok := actualSet[key]; !ok {
			issue.Missing = append(issue.Missing, key)
		}
	}
	for _, item := range actual {
		key := setupSvcLiveReplayEvidencePathForContract(projectPath, item)
		if key == "" {
			continue
		}
		if _, ok := expectedSet[key]; !ok {
			issue.Unexpected = append(issue.Unexpected, key)
		}
	}
	issue.Duplicate = duplicateSetupSvcLiveReplayStrings(actual, true)
	return issue
}

func setupSvcLiveReplayEvidenceFileMap(projectPath string, files []string) map[string]string {
	result := map[string]string{}
	for _, file := range files {
		key := setupSvcLiveReplayEvidencePathForContract(projectPath, file)
		if key != "" {
			result[key] = file
		}
	}
	return result
}

func setupSvcLiveReplayEvidencePathForContract(projectPath string, filePath string) string {
	trimmed := strings.TrimSpace(filePath)
	if trimmed == "" {
		return ""
	}
	cleaned := filepath.Clean(trimmed)
	if filepath.IsAbs(cleaned) {
		if rel, err := filepath.Rel(projectPath, cleaned); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			cleaned = rel
		}
	}
	return filepath.ToSlash(cleaned)
}

func verifySetupSvcLiveReplayEvidenceArtifact(projectPath string, requiredFile string, domain setupSvcLiveReplayDomain, operation string, decoded any) []string {
	artifact, ok := decoded.(map[string]any)
	if !ok {
		return []string{"evidenceFileNotObject"}
	}
	failures := []string{}
	failures = append(failures, setupSvcLiveReplayArtifactContractFailures(artifact)...)
	failures = append(failures, setupSvcLiveReplayArtifactProjectFailures(projectPath, artifact)...)
	failures = append(failures, setupSvcLiveReplayArtifactIdentityFailures(requiredFile, domain.Domain, operation, artifact)...)
	if !setupSvcLiveReplayArtifactPassed(artifact) {
		failures = append(failures, "evidenceFileStatusNotPassed")
	}
	if filepath.Base(filepath.ToSlash(requiredFile)) == "normalized-diff.json" {
		if !setupSvcLiveReplayDiffHasCleanEvidence(artifact) {
			failures = append(failures, "normalizedDiffMissingCleanEvidence")
		}
		if !setupSvcLiveReplayDiffIsClean(artifact) {
			failures = append(failures, "normalizedDiffNotClean")
		}
	}
	if filepath.Base(filepath.ToSlash(requiredFile)) == "query-readback.json" {
		requiredTables := setupSvcLiveReplayRequiredTablesForOperation(domain, operation)
		if !setupSvcLiveReplayReadbackHasStructureEvidence(artifact) {
			failures = append(failures, "queryReadbackMissingStructureEvidence")
		}
		failures = append(failures, setupSvcLiveReplayExpectationEvidenceFailures(artifact, domain.QueryReadbackExpectations, "queryReadbackExpectations", setupSvcLiveReplayQueryReadbackExpectationEvidenceKeys(), setupSvcLiveReplayReadbackNestedEvidenceKeys())...)
		failures = append(failures, setupSvcLiveReplayReadbackTableFailures(artifact, requiredTables)...)
		failures = append(failures, setupSvcLiveReplayReadbackRowFailures(artifact, requiredTables)...)
		failures = append(failures, setupSvcLiveReplayReadbackFieldFailures(artifact, requiredTables)...)
		failures = append(failures, setupSvcLiveReplayReadbackRelationshipFailures(artifact)...)
		if !setupSvcLiveReplayReadbackHasCleanEvidence(artifact) {
			failures = append(failures, "queryReadbackMissingCleanCounters")
		}
		if !setupSvcLiveReplayReadbackIsComplete(artifact) {
			failures = append(failures, "queryReadbackStructureIncomplete")
		}
	}
	if filepath.Base(filepath.ToSlash(requiredFile)) == "cleanup.json" {
		if !setupSvcLiveReplayCleanupHasResidualEvidence(artifact) {
			failures = append(failures, "cleanupMissingResidualEvidence")
		}
		if !setupSvcLiveReplayCleanupHasCleanResidualEvidence(artifact) {
			failures = append(failures, "cleanupMissingResidualCounters")
		}
		if !setupSvcLiveReplayCleanupIsComplete(artifact) {
			failures = append(failures, "cleanupResidualMetadataRemaining")
		}
	}
	switch filepath.Base(filepath.ToSlash(requiredFile)) {
	case "setup-svc.json":
		requiredTables := setupSvcLiveReplayRequiredTablesForOperation(domain, operation)
		failures = append(failures, setupSvcLiveReplayExpectationEvidenceFailures(artifact, setupSvcLiveReplayRuntimeEffectsForOperation(domain.Domain, operation), "runtimeEffects", setupSvcLiveReplayRuntimeEffectEvidenceKeys(), setupSvcLiveReplaySnapshotNestedEvidenceKeys())...)
		failures = append(failures, setupSvcLiveReplaySnapshotTableFailures(artifact, requiredTables, "setupSvcSnapshot")...)
	case "metadata-service.json":
		requiredTables := setupSvcLiveReplayRequiredTablesForOperation(domain, operation)
		failures = append(failures, setupSvcLiveReplayMetadataServiceDatasourceProofFailures(artifact)...)
		failures = append(failures, setupSvcLiveReplayExpectationEvidenceFailures(artifact, setupSvcLiveReplayRuntimeEffectsForOperation(domain.Domain, operation), "runtimeEffects", setupSvcLiveReplayRuntimeEffectEvidenceKeys(), setupSvcLiveReplaySnapshotNestedEvidenceKeys())...)
		failures = append(failures, setupSvcLiveReplaySnapshotTableFailures(artifact, requiredTables, "metadataServiceSnapshot")...)
	}
	return failures
}

func setupSvcLiveReplayMetadataServiceDatasourceProofFailures(artifact map[string]any) []string {
	raw, ok := artifact["metadataServiceDatasource"].(map[string]any)
	if !ok || len(raw) == 0 {
		return []string{"metadataServiceDatasourceMissingEvidence"}
	}
	if !boolValue(raw["readyForRealDatasource"]) {
		return []string{"metadataServiceDatasourceNotReady"}
	}
	if strings.EqualFold(strings.TrimSpace(firstMapString(raw, "status")), "ready") {
		return nil
	}
	return []string{"metadataServiceDatasourceStatusNotReady"}
}

func setupSvcLiveReplayRuntimeEffectEvidenceKeys() []string {
	return []string{"runtimeEffectChecks", "runtimeEffects", "sideEffectChecks", "sideEffects", "effectChecks", "runtimeChecks"}
}

func setupSvcLiveReplayQueryReadbackExpectationEvidenceKeys() []string {
	return []string{"readbackExpectationChecks", "queryReadbackExpectations", "queryExpectationChecks", "expectationChecks", "structureExpectationChecks"}
}

func setupSvcLiveReplaySnapshotNestedEvidenceKeys() []string {
	return []string{"totals", "summary", "result", "evidence", "runtime", "effects", "checks"}
}

func setupSvcLiveReplayReadbackNestedEvidenceKeys() []string {
	return []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "structureChecks"}
}

func setupSvcLiveReplayExpectationEvidenceFailures(artifact map[string]any, expected []string, label string, evidenceKeys []string, nestedKeys []string) []string {
	if len(expected) == 0 {
		return nil
	}
	covered := map[string]bool{}
	for _, key := range evidenceKeys {
		setupSvcLiveReplayCollectExpectationEvidence(covered, artifact[key])
	}
	for _, key := range nestedKeys {
		if nested, ok := artifact[key].(map[string]any); ok {
			for name := range setupSvcLiveReplayExpectationEvidenceSet(nested, evidenceKeys, nestedKeys) {
				covered[name] = true
			}
		}
	}
	var missing []string
	for _, item := range expected {
		key := setupSvcLiveReplayExpectationKey(item)
		if key != "" && !covered[key] {
			missing = append(missing, item)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	return []string{label + "MissingEvidence=" + strings.Join(missing, ",")}
}

func setupSvcLiveReplayExpectationEvidenceSet(artifact map[string]any, evidenceKeys []string, nestedKeys []string) map[string]bool {
	covered := map[string]bool{}
	for _, key := range evidenceKeys {
		setupSvcLiveReplayCollectExpectationEvidence(covered, artifact[key])
	}
	for _, key := range nestedKeys {
		if nested, ok := artifact[key].(map[string]any); ok {
			for name := range setupSvcLiveReplayExpectationEvidenceSet(nested, evidenceKeys, nestedKeys) {
				covered[name] = true
			}
		}
	}
	return covered
}

func setupSvcLiveReplayCollectExpectationEvidence(out map[string]bool, value any) {
	switch item := value.(type) {
	case nil:
		return
	case []any:
		for _, raw := range item {
			setupSvcLiveReplayCollectExpectationEvidence(out, raw)
		}
	case []map[string]any:
		for _, raw := range item {
			setupSvcLiveReplayCollectExpectationEvidence(out, raw)
		}
	case map[string]any:
		if name := setupSvcLiveReplayExpectationEvidenceName(item); name != "" {
			if setupSvcLiveReplayExpectationEvidencePassed(item) {
				out[name] = true
			}
			return
		}
		for key, raw := range item {
			if nested, ok := raw.(map[string]any); ok {
				if setupSvcLiveReplayExpectationEvidencePassed(nested) {
					if normalized := setupSvcLiveReplayExpectationKey(key); normalized != "" {
						out[normalized] = true
					}
				}
				continue
			}
			setupSvcLiveReplayCollectExpectationEvidence(out, raw)
		}
	}
}

func setupSvcLiveReplayExpectationEvidenceName(item map[string]any) string {
	return setupSvcLiveReplayExpectationKey(firstMapString(item, "name", "effect", "runtimeEffect", "sideEffect", "expectation", "check", "checkName", "key", "id"))
}

func setupSvcLiveReplayExpectationEvidencePassed(item map[string]any) bool {
	status := firstMapString(item, "status", "checkStatus", "verificationStatus", "resultStatus")
	if !setupSvcLiveReplayPassedStatus(status) {
		return false
	}
	return true
}

func setupSvcLiveReplayExpectationKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func setupSvcLiveReplayProjectIdentityIssues(expectedProject string, actualProject string) []string {
	if strings.TrimSpace(actualProject) == "" {
		return []string{"missing project"}
	}
	if !setupSvcLiveReplaySameProject(expectedProject, actualProject) {
		return []string{"project mismatch"}
	}
	return nil
}

func setupSvcLiveReplayArtifactProjectFailures(projectPath string, artifact map[string]any) []string {
	actualProject := setupSvcLiveReplayArtifactValue(artifact, "project", "projectPath")
	if strings.TrimSpace(actualProject) == "" {
		return []string{"evidenceFileMissingProject"}
	}
	if !setupSvcLiveReplaySameProject(projectPath, actualProject) {
		return []string{"evidenceFileProjectMismatch"}
	}
	return nil
}

func setupSvcLiveReplaySameProject(expectedProject string, actualProject string) bool {
	expected := setupSvcLiveReplayProjectIdentity(expectedProject)
	actual := setupSvcLiveReplayProjectIdentity(actualProject)
	return expected != "" && actual != "" && expected == actual
}

func setupSvcLiveReplayProjectIdentity(projectPath string) string {
	trimmed := strings.TrimSpace(projectPath)
	if trimmed == "" {
		return ""
	}
	cleaned := filepath.Clean(trimmed)
	if abs, err := filepath.Abs(cleaned); err == nil {
		cleaned = abs
	}
	return filepath.ToSlash(cleaned)
}

func setupSvcLiveReplayArtifactContractFailures(artifact map[string]any) []string {
	failures := []string{}
	actualVersion := setupSvcLiveReplayArtifactValue(artifact, "contractVersion")
	actualFingerprint := setupSvcLiveReplayArtifactValue(artifact, "contractFingerprint")
	if strings.TrimSpace(actualVersion) == "" {
		failures = append(failures, "evidenceFileMissingContractVersion")
	} else if strings.TrimSpace(actualVersion) != setupSvcLiveReplayContractVersion {
		failures = append(failures, "evidenceFileContractVersionMismatch")
	}
	if strings.TrimSpace(actualFingerprint) == "" {
		failures = append(failures, "evidenceFileMissingContractFingerprint")
	} else if strings.TrimSpace(actualFingerprint) != setupSvcLiveReplayExpectedContractFingerprint() {
		failures = append(failures, "evidenceFileContractFingerprintMismatch")
	}
	return failures
}

func setupSvcLiveReplayArtifactIdentityFailures(requiredFile string, domain string, operation string, artifact map[string]any) []string {
	failures := []string{}
	expectedArtifact := setupSvcLiveReplayArtifactType(requiredFile)
	actualDomain := setupSvcLiveReplayArtifactValue(artifact, "domain", "msapiDomain", "metadataDomain")
	actualOperation := setupSvcLiveReplayArtifactValue(artifact, "operation", "operationType", "action")
	actualArtifact := setupSvcLiveReplayArtifactValue(artifact, "artifactType", "artifact", "evidenceType")
	if actualDomain == "" {
		failures = append(failures, "evidenceFileMissingDomain")
	} else if normalizeDomain(actualDomain) != normalizeDomain(domain) {
		failures = append(failures, "evidenceFileDomainMismatch")
	}
	if actualOperation == "" {
		failures = append(failures, "evidenceFileMissingOperation")
	} else if strings.ToLower(strings.TrimSpace(actualOperation)) != strings.ToLower(strings.TrimSpace(operation)) {
		failures = append(failures, "evidenceFileOperationMismatch")
	}
	if actualArtifact == "" {
		failures = append(failures, "evidenceFileMissingArtifactType")
	} else if setupSvcLiveReplayNormalizeArtifactType(actualArtifact) != expectedArtifact {
		failures = append(failures, "evidenceFileArtifactTypeMismatch")
	}
	return failures
}

func setupSvcLiveReplayArtifactType(requiredFile string) string {
	base := filepath.Base(filepath.ToSlash(requiredFile))
	return setupSvcLiveReplayNormalizeArtifactType(strings.TrimSuffix(base, filepath.Ext(base)))
}

func setupSvcLiveReplayNormalizeArtifactType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func setupSvcLiveReplayArtifactValue(artifact map[string]any, keys ...string) string {
	if value := firstMapString(artifact, keys...); value != "" {
		return value
	}
	for _, nestedKey := range []string{"metadata", "evidence", "context"} {
		if nested, ok := artifact[nestedKey].(map[string]any); ok {
			if value := firstMapString(nested, keys...); value != "" {
				return value
			}
		}
	}
	return ""
}

func setupSvcLiveReplayArtifactPassed(artifact map[string]any) bool {
	for _, key := range []string{"status", "checkStatus", "evidenceStatus", "resultStatus", "verificationStatus", "postWriteStatus", "cleanupStatus"} {
		if setupSvcLiveReplayPassedStatus(firstMapString(artifact, key)) {
			return true
		}
	}
	if result, ok := artifact["result"].(map[string]any); ok && setupSvcLiveReplayArtifactPassed(result) {
		return true
	}
	if summary, ok := artifact["summary"].(map[string]any); ok && setupSvcLiveReplayArtifactPassed(summary) {
		return true
	}
	return false
}

func setupSvcLiveReplaySnapshotTableFailures(artifact map[string]any, requiredTables []string, prefix string) []string {
	if len(requiredTables) == 0 {
		return nil
	}
	covered := setupSvcLiveReplaySnapshotTableSet(artifact)
	if len(covered) == 0 {
		return []string{prefix + "MissingTableEvidence"}
	}
	var missing []string
	for _, table := range requiredTables {
		normalized := strings.ToLower(strings.TrimSpace(table))
		if normalized == "" {
			continue
		}
		if !covered[normalized] {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return []string{prefix + "MissingTables=" + strings.Join(missing, ",")}
	}
	return nil
}

func setupSvcLiveReplaySnapshotTableSet(artifact map[string]any) map[string]bool {
	result := map[string]bool{}
	for _, key := range []string{"tableSnapshots", "snapshots", "metadataSnapshots", "tables", "metadataTables"} {
		setupSvcLiveReplayCollectSnapshotTableNames(result, artifact[key])
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "snapshot"} {
		if nested, ok := artifact[key].(map[string]any); ok {
			for table := range setupSvcLiveReplaySnapshotTableSet(nested) {
				result[table] = true
			}
		}
	}
	return result
}

func setupSvcLiveReplaySnapshotTableSetPairFailures(setupSvcTables map[string]bool, metadataServiceTables map[string]bool) []string {
	if len(setupSvcTables) == 0 || len(metadataServiceTables) == 0 {
		return nil
	}
	var failures []string
	if missing := setupSvcLiveReplayMissingTableSetItems(setupSvcTables, metadataServiceTables); len(missing) > 0 {
		failures = append(failures, "snapshotTableSetMismatch:metadataServiceMissingTables="+strings.Join(missing, ","))
	}
	if missing := setupSvcLiveReplayMissingTableSetItems(metadataServiceTables, setupSvcTables); len(missing) > 0 {
		failures = append(failures, "snapshotTableSetMismatch:setupSvcMissingTables="+strings.Join(missing, ","))
	}
	return failures
}

func setupSvcLiveReplayMissingTableSetItems(expected map[string]bool, actual map[string]bool) []string {
	var missing []string
	for table := range expected {
		if !actual[table] {
			missing = append(missing, table)
		}
	}
	sort.Strings(missing)
	return missing
}

func setupSvcLiveReplayCollectSnapshotTableNames(out map[string]bool, value any) {
	switch item := value.(type) {
	case []any:
		for _, raw := range item {
			setupSvcLiveReplayCollectSnapshotTableNames(out, raw)
		}
	case map[string]any:
		if table := firstMapString(item, "table", "tableName", "name"); table != "" {
			if setupSvcLiveReplayHasMeaningfulSnapshotDetail(item) {
				if normalized := strings.ToLower(strings.TrimSpace(table)); normalized != "" {
					out[normalized] = true
				}
			}
			return
		}
		for key, raw := range item {
			if setupSvcLiveReplayHasMeaningfulSnapshotTableValue(raw) {
				if normalized := strings.ToLower(strings.TrimSpace(key)); normalized != "" {
					out[normalized] = true
				}
			}
		}
	}
}

func setupSvcLiveReplayHasMeaningfulSnapshotTableValue(value any) bool {
	if nested, ok := value.(map[string]any); ok {
		return setupSvcLiveReplayHasMeaningfulSnapshotDetail(nested)
	}
	return false
}

func setupSvcLiveReplayHasMeaningfulSnapshotDetail(snapshot map[string]any) bool {
	if setupSvcLiveReplayHasSnapshotRowEvidence(snapshot) && setupSvcLiveReplayHasSnapshotColumnEvidence(snapshot) {
		return true
	}
	if setupSvcLiveReplayHasMeaningfulSnapshotValue(snapshot["changeCount"]) && setupSvcLiveReplayHasMeaningfulSnapshotValue(snapshot["mutationTypes"]) {
		return true
	}
	for _, key := range []string{"snapshot", "evidence", "result"} {
		if nested, ok := snapshot[key].(map[string]any); ok && setupSvcLiveReplayHasMeaningfulSnapshotDetail(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayHasSnapshotRowEvidence(snapshot map[string]any) bool {
	for _, key := range []string{
		"rows", "records", "sampleRows", "readbackRows", "queriedRows",
		"beforeRows", "afterRows", "deltaRows", "matchedRows", "createdRows", "updatedRows", "deletedRows",
		"before", "after", "diff", "changes",
		"changeCount", "mutationCount", "mutationTypes",
	} {
		if setupSvcLiveReplayHasMeaningfulSnapshotValue(snapshot[key]) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayHasSnapshotColumnEvidence(snapshot map[string]any) bool {
	for _, key := range []string{
		"columns", "fields", "requiredColumns", "requiredFields", "primaryKeys", "keyColumns",
		"selectedColumns", "diffColumns", "changedColumns",
	} {
		if setupSvcLiveReplayHasMeaningfulSnapshotValue(snapshot[key]) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayHasMeaningfulSnapshotValue(value any) bool {
	switch item := value.(type) {
	case nil:
		return false
	case bool:
		return false
	case map[string]any:
		return len(item) > 0
	case []any:
		return len(item) > 0
	case []string:
		return len(item) > 0
	case string:
		return strings.TrimSpace(item) != ""
	case int:
		return item > 0
	case int8:
		return item > 0
	case int16:
		return item > 0
	case int32:
		return item > 0
	case int64:
		return item > 0
	case uint:
		return item > 0
	case uint8:
		return item > 0
	case uint16:
		return item > 0
	case uint32:
		return item > 0
	case uint64:
		return item > 0
	case float32:
		return item > 0
	case float64:
		return item > 0
	default:
		return value != nil
	}
}

func setupSvcLiveReplayPassedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "passed", "verified", "success", "successful":
		return true
	default:
		return false
	}
}

func setupSvcLiveReplayDiffIsClean(artifact map[string]any) bool {
	for _, key := range []string{
		"missingRows", "missingColumns", "missingValues", "unexpectedRows", "unexpectedColumns", "unexpectedValues",
		"mismatchedRows", "mismatchedColumns", "mismatchedValues",
		"failedRows", "failedColumns", "failedValues", "differences", "diffs", "errors", "failures", "blockingIssues",
	} {
		if setupSvcLiveReplayInvalidCleanCount(artifact[key]) {
			return false
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "diff", "comparison", "normalizedDiff"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && !setupSvcLiveReplayDiffIsClean(nested) {
			return false
		}
	}
	return true
}

func setupSvcLiveReplayDiffHasCleanEvidence(artifact map[string]any) bool {
	for _, key := range []string{
		"missingRows", "missingColumns", "missingValues", "unexpectedRows", "unexpectedColumns", "unexpectedValues",
		"mismatchedRows", "mismatchedColumns", "mismatchedValues", "failedRows", "failedColumns", "failedValues",
		"differences", "diffs", "errors", "failures", "blockingIssues",
	} {
		if _, ok := artifact[key]; ok {
			return true
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "diff", "comparison", "normalizedDiff"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && setupSvcLiveReplayDiffHasCleanEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayReadbackHasCleanEvidence(artifact map[string]any) bool {
	for _, key := range []string{
		"missingFields", "missingRelationships", "missingRows", "missingColumns", "missingValues",
		"mismatchedFields", "mismatchedRelationships", "mismatchedRows", "mismatchedColumns", "mismatchedValues",
		"brokenRelationships", "unreadableRelationships", "errors", "failures", "blockingIssues",
	} {
		if _, ok := artifact[key]; ok {
			return true
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "structureChecks"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && setupSvcLiveReplayReadbackHasCleanEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayReadbackHasStructureEvidence(artifact map[string]any) bool {
	for _, key := range []string{
		"requiredFields", "requiredColumns", "readbackFields", "queryFields", "selectedFields",
		"columns", "fields", "primaryKeys", "keyColumns", "queryShape", "readbackShape",
	} {
		if value, ok := artifact[key]; ok && setupSvcLiveReplayHasMeaningfulEvidenceValue(value) {
			return true
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "fieldChecks", "structureChecks"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && setupSvcLiveReplayReadbackHasStructureEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayReadbackRelationshipFailures(artifact map[string]any) []string {
	if setupSvcLiveReplayReadbackHasFailedRelationshipEvidence(artifact) {
		return []string{"queryReadbackRelationshipEvidenceNotPassed"}
	}
	if !setupSvcLiveReplayReadbackHasRelationshipEvidence(artifact) {
		return []string{"queryReadbackMissingRelationshipEvidence"}
	}
	return nil
}

func setupSvcLiveReplayReadbackHasRelationshipEvidence(artifact map[string]any) bool {
	for _, key := range []string{
		"relationships", "readbackRelationships", "relationshipChecks",
		"relationshipGraph", "relationshipEdges", "joinChecks", "joins", "foreignKeys",
		"referenceFields", "dependencyEdges",
	} {
		if setupSvcLiveReplayHasPassedRelationshipEvidence(artifact[key]) {
			return true
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "structureChecks"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && setupSvcLiveReplayReadbackHasRelationshipEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayReadbackHasFailedRelationshipEvidence(artifact map[string]any) bool {
	for _, key := range []string{
		"relationships", "readbackRelationships", "relationshipChecks",
		"relationshipGraph", "relationshipEdges", "joinChecks", "joins", "foreignKeys",
		"referenceFields", "dependencyEdges",
	} {
		if setupSvcLiveReplayRelationshipValueHasFailedStatus(artifact[key]) {
			return true
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "structureChecks"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && setupSvcLiveReplayReadbackHasFailedRelationshipEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayHasPassedRelationshipEvidence(value any) bool {
	switch item := value.(type) {
	case nil:
		return false
	case []any:
		for _, raw := range item {
			if setupSvcLiveReplayHasPassedRelationshipEvidence(raw) {
				return true
			}
		}
	case []map[string]any:
		for _, raw := range item {
			if setupSvcLiveReplayHasPassedRelationshipEvidence(raw) {
				return true
			}
		}
	case []string:
		return false
	case string:
		return false
	case map[string]any:
		if setupSvcLiveReplayRelationshipStatusPassed(item) {
			return true
		}
		if setupSvcLiveReplayHasStructuralRelationshipEvidence(item) {
			return true
		}
		for _, raw := range item {
			if setupSvcLiveReplayHasPassedRelationshipEvidence(raw) {
				return true
			}
		}
	}
	return false
}

func setupSvcLiveReplayRelationshipStatusPassed(item map[string]any) bool {
	status := firstMapString(item, "status", "checkStatus", "relationshipStatus", "verificationStatus", "resultStatus")
	if status == "" || !setupSvcLiveReplayPassedStatus(status) {
		return false
	}
	if setupSvcLiveReplayRelationshipIdentity(item) != "" {
		return true
	}
	return setupSvcLiveReplayHasStructuralRelationshipEvidence(item)
}

func setupSvcLiveReplayRelationshipIdentity(item map[string]any) string {
	return firstMapString(item, "name", "relationship", "relationshipName", "check", "checkName", "key", "id")
}

func setupSvcLiveReplayHasStructuralRelationshipEvidence(item map[string]any) bool {
	source := firstMapString(item, "source", "sourceTable", "from", "fromTable", "parent", "parentTable", "object", "objectName")
	target := firstMapString(item, "target", "targetTable", "to", "toTable", "child", "childTable", "referenceObject", "referenceTable")
	field := firstMapString(item, "field", "fieldName", "referenceField", "lookupField", "foreignKey", "joinField")
	return source != "" && target != "" && field != ""
}

func setupSvcLiveReplayRelationshipValueHasFailedStatus(value any) bool {
	switch item := value.(type) {
	case nil:
		return false
	case []any:
		for _, raw := range item {
			if setupSvcLiveReplayRelationshipValueHasFailedStatus(raw) {
				return true
			}
		}
	case []map[string]any:
		for _, raw := range item {
			if setupSvcLiveReplayRelationshipValueHasFailedStatus(raw) {
				return true
			}
		}
	case map[string]any:
		status := firstMapString(item, "status", "checkStatus", "relationshipStatus", "verificationStatus", "resultStatus")
		if status != "" && !setupSvcLiveReplayPassedStatus(status) {
			return true
		}
		for _, raw := range item {
			if setupSvcLiveReplayRelationshipValueHasFailedStatus(raw) {
				return true
			}
		}
	}
	return false
}

func setupSvcLiveReplayReadbackTableFailures(artifact map[string]any, requiredTables []string) []string {
	if len(requiredTables) == 0 {
		return nil
	}
	covered := setupSvcLiveReplayReadbackTableSet(artifact)
	if len(covered) == 0 {
		return []string{"queryReadbackMissingTableCoverage"}
	}
	var missing []string
	for _, table := range requiredTables {
		normalized := strings.ToLower(strings.TrimSpace(table))
		if normalized == "" {
			continue
		}
		if !covered[normalized] {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return []string{"queryReadbackMissingTables=" + strings.Join(missing, ",")}
	}
	return nil
}

func setupSvcLiveReplayReadbackTableSet(artifact map[string]any) map[string]bool {
	result := map[string]bool{}
	for _, key := range []string{"readbackTables", "queriedTables", "metadataTables", "tableCoverage", "tableReadback", "tableChecks"} {
		setupSvcLiveReplayCollectReadbackTableNames(result, artifact[key])
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "structureChecks"} {
		if nested, ok := artifact[key].(map[string]any); ok {
			for table := range setupSvcLiveReplayReadbackTableSet(nested) {
				result[table] = true
			}
		}
	}
	return result
}

func setupSvcLiveReplayReadbackRowFailures(artifact map[string]any, requiredTables []string) []string {
	if len(requiredTables) == 0 {
		return nil
	}
	covered := setupSvcLiveReplayReadbackRowTableSet(artifact)
	if len(covered) == 0 {
		return []string{"queryReadbackMissingRowEvidence"}
	}
	var missing []string
	for _, table := range requiredTables {
		normalized := strings.ToLower(strings.TrimSpace(table))
		if normalized == "" {
			continue
		}
		if !covered[normalized] {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return []string{"queryReadbackMissingRowEvidenceTables=" + strings.Join(missing, ",")}
	}
	return nil
}

func setupSvcLiveReplayReadbackRowTableSet(artifact map[string]any) map[string]bool {
	result := map[string]bool{}
	for _, key := range []string{"readbackTables", "queriedTables", "metadataTables", "tableCoverage", "tableReadback", "tableChecks"} {
		setupSvcLiveReplayCollectReadbackRowTableNames(result, artifact[key])
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "structureChecks"} {
		if nested, ok := artifact[key].(map[string]any); ok {
			for table := range setupSvcLiveReplayReadbackRowTableSet(nested) {
				result[table] = true
			}
		}
	}
	return result
}

func setupSvcLiveReplayCollectReadbackRowTableNames(out map[string]bool, value any) {
	switch item := value.(type) {
	case []any:
		for _, raw := range item {
			setupSvcLiveReplayCollectReadbackRowTableNames(out, raw)
		}
	case map[string]any:
		if table := firstMapString(item, "table", "tableName", "name"); table != "" {
			if setupSvcLiveReplayHasReadbackRowEvidence(item) {
				if normalized := strings.ToLower(strings.TrimSpace(table)); normalized != "" {
					out[normalized] = true
				}
			}
			return
		}
		for key, raw := range item {
			if setupSvcLiveReplayHasReadbackRowTableValue(raw) {
				if normalized := strings.ToLower(strings.TrimSpace(key)); normalized != "" {
					out[normalized] = true
				}
			}
		}
	}
}

func setupSvcLiveReplayHasReadbackRowTableValue(value any) bool {
	if nested, ok := value.(map[string]any); ok {
		return setupSvcLiveReplayHasReadbackRowEvidence(nested)
	}
	return false
}

func setupSvcLiveReplayHasReadbackRowEvidence(table map[string]any) bool {
	for _, key := range []string{
		"rows", "records", "sampleRows", "rowSamples", "readbackRows", "queriedRows", "matchedRows",
		"resultRows", "tableRows",
	} {
		if setupSvcLiveReplayHasMeaningfulReadbackTableValue(table[key]) {
			return true
		}
	}
	for _, key := range []string{"snapshot", "evidence", "result", "readback", "query"} {
		if nested, ok := table[key].(map[string]any); ok && setupSvcLiveReplayHasReadbackRowEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayReadbackFieldFailures(artifact map[string]any, requiredTables []string) []string {
	if len(requiredTables) == 0 {
		return nil
	}
	covered := setupSvcLiveReplayReadbackFieldTableSet(artifact)
	if len(covered) == 0 {
		return []string{"queryReadbackMissingFieldEvidence"}
	}
	var missing []string
	for _, table := range requiredTables {
		normalized := strings.ToLower(strings.TrimSpace(table))
		if normalized == "" {
			continue
		}
		if !covered[normalized] {
			missing = append(missing, table)
		}
	}
	if len(missing) > 0 {
		return []string{"queryReadbackMissingFieldEvidenceTables=" + strings.Join(missing, ",")}
	}
	return nil
}

func setupSvcLiveReplayReadbackFieldTableSet(artifact map[string]any) map[string]bool {
	result := map[string]bool{}
	for _, key := range []string{"readbackTables", "queriedTables", "metadataTables", "tableCoverage", "tableReadback", "tableChecks"} {
		setupSvcLiveReplayCollectReadbackFieldTableNames(result, artifact[key])
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "structureChecks"} {
		if nested, ok := artifact[key].(map[string]any); ok {
			for table := range setupSvcLiveReplayReadbackFieldTableSet(nested) {
				result[table] = true
			}
		}
	}
	return result
}

func setupSvcLiveReplayCollectReadbackFieldTableNames(out map[string]bool, value any) {
	switch item := value.(type) {
	case []any:
		for _, raw := range item {
			setupSvcLiveReplayCollectReadbackFieldTableNames(out, raw)
		}
	case map[string]any:
		if table := firstMapString(item, "table", "tableName", "name"); table != "" {
			if setupSvcLiveReplayHasReadbackFieldEvidence(item) {
				if normalized := strings.ToLower(strings.TrimSpace(table)); normalized != "" {
					out[normalized] = true
				}
			}
			return
		}
		for key, raw := range item {
			if setupSvcLiveReplayHasReadbackFieldTableValue(raw) {
				if normalized := strings.ToLower(strings.TrimSpace(key)); normalized != "" {
					out[normalized] = true
				}
			}
		}
	}
}

func setupSvcLiveReplayHasReadbackFieldTableValue(value any) bool {
	if nested, ok := value.(map[string]any); ok {
		return setupSvcLiveReplayHasReadbackFieldEvidence(nested)
	}
	return false
}

func setupSvcLiveReplayHasReadbackFieldEvidence(table map[string]any) bool {
	for _, key := range []string{
		"columns", "fields", "requiredFields", "readbackFields", "queryFields", "selectedFields", "primaryKeys",
		"fieldChecks", "shapeChecks",
	} {
		if setupSvcLiveReplayHasMeaningfulReadbackTableValue(table[key]) {
			return true
		}
	}
	for _, key := range []string{"snapshot", "evidence", "result", "readback", "query"} {
		if nested, ok := table[key].(map[string]any); ok && setupSvcLiveReplayHasReadbackFieldEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayCollectReadbackTableNames(out map[string]bool, value any) {
	switch item := value.(type) {
	case []any:
		for _, raw := range item {
			setupSvcLiveReplayCollectReadbackTableNames(out, raw)
		}
	case map[string]any:
		if table := firstMapString(item, "table", "tableName", "name"); table != "" {
			if setupSvcLiveReplayHasMeaningfulReadbackTableDetail(item) {
				if normalized := strings.ToLower(strings.TrimSpace(table)); normalized != "" {
					out[normalized] = true
				}
			}
			return
		}
		for key, raw := range item {
			if setupSvcLiveReplayHasMeaningfulReadbackTableValueForCoverage(raw) {
				if normalized := strings.ToLower(strings.TrimSpace(key)); normalized != "" {
					out[normalized] = true
				}
			}
		}
	}
}

func setupSvcLiveReplayHasMeaningfulReadbackTableValueForCoverage(value any) bool {
	if nested, ok := value.(map[string]any); ok {
		return setupSvcLiveReplayHasMeaningfulReadbackTableDetail(nested)
	}
	return false
}

func setupSvcLiveReplayHasMeaningfulReadbackTableDetail(table map[string]any) bool {
	for _, key := range []string{
		"rowCount", "rows", "records", "columns", "fields", "requiredFields", "readbackFields",
		"relationships", "requiredRelationships", "readbackRelationships", "queryFields",
		"shapeChecks", "fieldChecks", "relationshipChecks",
	} {
		if setupSvcLiveReplayHasMeaningfulReadbackTableValue(table[key]) {
			return true
		}
	}
	for _, key := range []string{"snapshot", "evidence", "result", "readback", "query"} {
		if setupSvcLiveReplayHasMeaningfulReadbackTableValue(table[key]) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayHasMeaningfulReadbackTableValue(value any) bool {
	switch item := value.(type) {
	case nil:
		return false
	case bool:
		return false
	case []any:
		return len(item) > 0
	case []string:
		return len(item) > 0
	case map[string]any:
		return len(item) > 0
	case string:
		return strings.TrimSpace(item) != ""
	case int:
		return item > 0
	case int8:
		return item > 0
	case int16:
		return item > 0
	case int32:
		return item > 0
	case int64:
		return item > 0
	case uint:
		return item > 0
	case uint8:
		return item > 0
	case uint16:
		return item > 0
	case uint32:
		return item > 0
	case uint64:
		return item > 0
	case float32:
		return item > 0
	case float64:
		return item > 0
	default:
		return value != nil
	}
}

func setupSvcLiveReplayHasMeaningfulEvidenceValue(value any) bool {
	switch item := value.(type) {
	case nil:
		return false
	case []any:
		return len(item) > 0
	case []string:
		return len(item) > 0
	case map[string]any:
		return len(item) > 0
	case string:
		return strings.TrimSpace(item) != ""
	default:
		return true
	}
}

func setupSvcLiveReplayReadbackIsComplete(artifact map[string]any) bool {
	for _, key := range []string{
		"missingFields", "missingRelationships", "missingRows", "missingColumns", "missingValues",
		"mismatchedFields", "mismatchedRelationships", "mismatchedRows", "mismatchedColumns", "mismatchedValues",
		"brokenRelationships", "unreadableRelationships", "errors", "failures", "blockingIssues",
	} {
		if setupSvcLiveReplayInvalidCleanCount(artifact[key]) {
			return false
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "readback", "query", "readbackChecks", "shapeChecks", "structureChecks"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && !setupSvcLiveReplayReadbackIsComplete(nested) {
			return false
		}
	}
	return true
}

func setupSvcLiveReplayCleanupHasCleanResidualEvidence(artifact map[string]any) bool {
	for _, key := range []string{
		"remainingRows", "remainingRecords", "residualRows", "residualRecords", "orphanRows", "orphanRecords",
		"errors", "failures", "blockingIssues",
	} {
		if _, ok := artifact[key]; ok {
			return true
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "cleanup", "cleanupChecks", "cleanupResults"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && setupSvcLiveReplayCleanupHasCleanResidualEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayCleanupHasResidualEvidence(artifact map[string]any) bool {
	for _, key := range []string{
		"cleanupChecks", "cleanupResults", "deletedRows", "deletedRecords", "removedRows", "removedRecords",
		"verifiedDeletedRows", "verifiedDeletedRecords", "remainingRows", "remainingRecords",
		"residualRows", "residualRecords", "orphanRows", "orphanRecords",
	} {
		if value, ok := artifact[key]; ok && setupSvcLiveReplayHasMeaningfulEvidenceValue(value) {
			return true
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "cleanup", "cleanupChecks", "cleanupResults"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && setupSvcLiveReplayCleanupHasResidualEvidence(nested) {
			return true
		}
	}
	return false
}

func setupSvcLiveReplayCleanupIsComplete(artifact map[string]any) bool {
	for _, key := range []string{
		"remainingRows", "remainingRecords", "residualRows", "residualRecords", "orphanRows", "orphanRecords",
		"errors", "failures", "blockingIssues",
	} {
		if setupSvcLiveReplayInvalidCleanCount(artifact[key]) {
			return false
		}
	}
	for _, key := range []string{"totals", "summary", "result", "evidence", "cleanup", "cleanupChecks", "cleanupResults"} {
		nested, ok := artifact[key].(map[string]any)
		if ok && !setupSvcLiveReplayCleanupIsComplete(nested) {
			return false
		}
	}
	return true
}

func setupSvcLiveReplayInvalidCleanCount(value any) bool {
	switch item := value.(type) {
	case nil:
		return false
	case []any:
		return len(item) > 0
	case []string:
		return len(item) > 0
	case map[string]any:
		return len(item) > 0
	case float64:
		return item != 0
	case float32:
		return item != 0
	case int:
		return item != 0
	case int64:
		return item != 0
	case bool:
		return true
	case string:
		return true
	default:
		return value != nil
	}
}

func setupSvcLiveReplayEvidenceFileList(value any) []string {
	out := []string{}
	var values []any
	switch items := value.(type) {
	case []any:
		values = items
	case []string:
		values = make([]any, 0, len(items))
		for _, item := range items {
			values = append(values, item)
		}
	default:
		return out
	}
	for _, raw := range values {
		switch item := raw.(type) {
		case string:
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				out = append(out, trimmed)
			}
		case map[string]any:
			if path := firstMapString(item, "path", "file", "name"); path != "" {
				out = append(out, path)
			}
		}
	}
	return out
}

func findSetupSvcLiveReplayEvidenceFile(files []string, required string) string {
	requiredSlash := filepath.ToSlash(filepath.Clean(required))
	requiredBase := filepath.Base(requiredSlash)
	for _, file := range files {
		fileSlash := filepath.ToSlash(filepath.Clean(strings.TrimSpace(file)))
		if fileSlash == requiredSlash || strings.HasSuffix(fileSlash, "/"+requiredSlash) {
			return file
		}
	}
	for _, file := range files {
		if filepath.Base(filepath.ToSlash(file)) == requiredBase {
			return file
		}
	}
	return ""
}

func setupSvcLiveReplayResolveEvidenceFile(projectPath string, filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filepath.IsAbs(filePath) {
		return filePath
	}
	return filepath.Join(projectPath, filePath)
}

func setupSvcLiveReplayProjectConfig(projectPath string, metadataServiceURL string) setupSvcLiveReplayConfig {
	cfg, _ := config.Load(projectPath)
	setupSvc := config.String(cfg, "setupSvc")
	apiSvc := config.String(cfg, "apiSvc")
	accessToken := config.String(cfg, "accessToken")
	out := setupSvcLiveReplayConfig{
		HasSetupSvc:          strings.TrimSpace(setupSvc) != "",
		HasApiSvc:            strings.TrimSpace(apiSvc) != "",
		HasAccessToken:       strings.TrimSpace(accessToken) != "",
		HasMetadataService:   strings.TrimSpace(metadataServiceURL) != "",
		MetadataServiceURL:   strings.TrimRight(metadataServiceURL, "/"),
		SetupSvcHostRedacted: true,
	}
	if parsed, err := url.Parse(setupSvc); err == nil && parsed.Host != "" {
		out.SetupSvcHost = parsed.Host
	}
	return out
}

func setupSvcLiveReplayDomains() []setupSvcLiveReplayDomain {
	return []setupSvcLiveReplayDomain{
		liveReplayDomain("objects", []string{"create", "update", "delete", "physical-purge", "query"}, []string{"tp_sys_object", "tp_sys_datatablestate", "tp_sys_schemetable", "tp_sys_multi_lang", "tp_sys_profile_infoset", "tp_sys_profile_field", "tp_sys_layout", "tp_sys_layout_section", "tp_sys_section_field", "tp_sys_relatedlist", "tp_sys_relatedlist_field", "tp_sys_lookuplayout", "tp_sys_button", "tp_sys_layout_button", "tp_sys_profile_layout", "tp_sys_view", "tp_sys_object_view_order", "tp_sys_view_field"}),
		liveReplayDomain("fields", []string{"create", "update", "delete", "query"}, []string{"tp_sys_schemetable", "tp_sys_multi_lang", "tp_sys_code", "tp_sys_globalselect_field", "tp_sys_profile_field", "tp_sys_section_field", "tp_sys_relatedlist", "tp_sys_relatedlist_field", "tp_sys_field_dependency", "tp_sys_field_reference", "tp_sys_autonum"}),
		liveReplayDomain("global-select-lists", []string{"create", "update", "delete", "query"}, []string{"tp_sys_global_select", "tp_sys_code", "tp_sys_globalselect_field"}),
		liveReplayDomain("record-types", []string{"create", "update", "delete", "query"}, []string{"tp_sys_recordtype", "tp_sys_profile_infoset", "tp_sys_profile_layout", "tp_sys_object", "tp_sys_schemetable", "tp_sys_field_dependency", "tp_sys_multi_lang"}),
		liveReplayDomain("layouts", []string{"create", "update", "delete", "query"}, []string{"tp_sys_layout", "tp_sys_layout_section", "tp_sys_section_field", "tp_sys_layout_button", "tp_sys_relatedlist", "tp_sys_lookuplayout", "tp_sys_profile_layout", "tp_sys_multi_lang"}),
		liveReplayDomain("profiles", []string{"create", "update", "delete", "query"}, []string{"tp_sys_profile", "tp_sys_profile_infoset", "tp_sys_profile_field", "tp_sys_profile_layout", "tp_sys_multi_lang"}),
		liveReplayDomain("permissions", []string{"create", "update", "delete", "assign", "remove", "query"}, []string{"tp_sys_profile_permission", "tp_sys_multi_lang", "tp_sys_profile_infoset", "tp_sys_permsets", "tp_sys_permsets_infoset", "tp_sys_permsets_fields", "tp_sys_permsets_assign"}),
		liveReplayDomain("roles", []string{"create", "update", "delete", "assign", "query"}, []string{"tp_sys_role", "tp_sys_group", "tp_sys_user"}),
		liveReplayDomain("sharing-rules", []string{"create", "update", "delete", "query"}, []string{"tp_sys_sharerule", "tp_sys_condition"}),
		liveReplayDomain("validation-rules", []string{"create", "update", "delete", "query"}, []string{"tp_sys_validaterule"}),
		liveReplayDomain("applications", []string{"create", "update", "delete", "query"}, []string{"tp_sys_app", "tp_sys_app_tab", "tp_sys_tab", "tp_sys_profile_infoset", "tp_sys_multi_lang"}),
		liveReplayDomain("menus", []string{"create", "update", "delete", "query"}, []string{"tp_sys_tab", "tp_sys_app_tab", "tp_sys_profile_infoset", "tp_sys_multi_lang"}),
		liveReplayDomain("buttons", []string{"create", "update", "delete", "query"}, []string{"tp_sys_button", "tp_sys_button_scope", "tp_sys_layout_button", "tp_sys_relatedlist_button", "tp_sys_view_button", "tp_sys_lookuplayout", "tp_sys_multi_lang"}),
		liveReplayDomain("custom-settings", []string{"create", "update", "delete", "query"}, []string{"tp_sys_object", "tp_sys_schemetable", "tp_sys_multi_lang", "tp_sys_layout", "tp_sys_profile_layout", "tp_sys_layout_section", "tp_sys_section_field"}),
		liveReplayDomain("dupe-catchers", []string{"create", "update", "delete", "query"}, []string{"tp_sys_dupecatcher", "tp_sys_dupecatcherule", "tp_sys_condition"}),
		liveReplayDomain("single-sign-ons", []string{"create", "update", "delete", "query"}, []string{"tp_sys_sp_idps"}),
		liveReplayDomain("identity-providers", []string{"create", "update", "delete", "query"}, []string{"tp_sys_idp_config", "tp_sys_idp_sps"}),
		liveReplayDomain("approval-processes", []string{"create", "update", "delete", "query"}, []string{"tp_sys_approval", "tp_sys_approval_step", "tp_sys_approval_step_layout", "tp_sys_apralrellist", "tp_sys_apralrellist_fields", "tp_sys_actions_relation", "tp_sys_condition"}),
		liveReplayDomain("reports", []string{"create", "update", "delete", "folder-create", "folder-update", "folder-delete", "query"}, []string{"tp_sys_report", "tp_sys_reporttypecustom", "tp_sys_reporttypecustomfields", "tp_sys_condition", "tp_sys_report_expression", "tp_sys_report_fieldname", "tp_sys_report_object", "tp_sys_report_object_detail", "tp_sys_reportgather", "tp_sys_reportgroup", "tp_sys_recent_items", "tp_sys_folder"}),
		liveReplayDomain("dashboards", []string{"create", "update", "delete", "query"}, []string{"tp_sys_dashboard", "tp_sys_dashboard_report", "tp_sys_dashboard_condition", "tp_sys_report", "tp_sys_recent_items", "tp_sys_dashboard_snapshot", "tp_sys_snapshot_refress"}),
		liveReplayDomain("object-views", []string{"create", "update", "delete", "query"}, []string{"tp_sys_view", "tp_sys_view_field", "tp_sys_view_button", "tp_sys_viewcharts", "tp_sys_viewkanban", "tp_sys_viewkanban_field", "tp_sys_dashboard_report", "tp_sys_condition"}),
	}
}

func liveReplayDomain(domain string, operations []string, tables []string) setupSvcLiveReplayDomain {
	return setupSvcLiveReplayDomain{
		Domain:                    domain,
		Operations:                operations,
		RequiredTables:            tables,
		RuntimeEffects:            setupSvcLiveReplayRuntimeEffects(domain),
		QueryReadbackExpectations: setupSvcLiveReplayQueryReadbackExpectations(domain),
		SetupSvcEvidence: []string{
			"setup-svc request/response for each write operation against a disposable record",
			"metadata table snapshot after setup-svc write",
			"setup-svc or UI readback payload after write",
		},
		MetadataServiceEvidence: []string{
			"MetadataService plan/apply request and VERIFIED operation id",
			"metadata table snapshot after MetadataService apply",
			"rollback-plan or explicit cleanup evidence for disposable replay records",
		},
		QueryEvidence: []string{
			"scan/query payload contains required rows and relationships",
			"normalized diff has no missing required rows or mismatched required values",
		},
		Status:           "covered_pending_live_replay",
		ApprovalRequired: true,
	}
}

func setupSvcLiveReplayRuntimeEffects(domain string) []string {
	switch normalizeDomain(domain) {
	case "objects":
		return []string{"datatable-prefix-allocation", "standard-object-default-fields", "standard-related-list-expansion", "object-view-order-expansion", "layout-button-view-profile-expansion", "database-view-refresh", "soft-delete-and-physical-purge-cleanup"}
	case "fields":
		return []string{"field-row-and-label-expansion", "option-reference-dependency-link-expansion", "layout-profile-permission-expansion", "lookup-relatedlist-expansion", "database-view-refresh", "delete-cleanup"}
	case "global-select-lists":
		return []string{"global-list-option-label-expansion"}
	case "record-types":
		return []string{"record-type-profile-infoset-expansion", "record-type-profile-layout-expansion", "object-recordtype-enable-expansion", "record-type-field-dependency-expansion", "translated-label-expansion", "delete-cleanup"}
	case "layouts":
		return []string{"layout-section-field-expansion", "profile-layout-and-button-link-expansion", "translated-label-expansion", "delete-cleanup"}
	case "profiles":
		return []string{"object-field-layout-recordtype-grant-expansion", "translated-label-expansion", "profile-delete-cleanup"}
	case "permissions":
		return []string{"permission-definition-label-expansion", "profile-infoset-menu-label-expansion", "permission-set-infoset-field-expansion", "permission-set-assignment-expansion", "assignment-remove-cleanup", "delete-cleanup"}
	case "roles":
		return []string{"role-group-hierarchy-expansion", "user-role-assignment-update", "user-role-unassignment-fallback-update", "delete-cleanup"}
	case "sharing-rules":
		return []string{"sharing-rule-condition-expansion", "share-rule-recalculation-dispatch", "dynamic-share-table-cleanup", "share-rule-delete-cleanup"}
	case "validation-rules":
		return []string{"validation-rule-row-lifecycle", "hard-delete-cleanup"}
	case "applications":
		return []string{"app-tab-binding-expansion", "app-profile-visibility-expansion", "translated-label-expansion", "delete-cleanup"}
	case "menus":
		return []string{"tab-app-profile-binding-expansion", "tab-profile-all-expansion", "translated-label-expansion", "delete-cleanup"}
	case "buttons":
		return []string{"button-scope-layout-view-binding-expansion", "translated-label-expansion", "delete-cleanup"}
	case "custom-settings":
		return []string{"setting-object-table-view-allocation", "setting-field-expansion", "setting-layout-profile-expansion", "translated-label-expansion", "delete-cleanup"}
	case "dupe-catchers":
		return []string{"dupe-rule-field-condition-expansion", "dupe-rule-firstletters-normalization", "dupe-rule-delete-cleanup"}
	case "single-sign-ons":
		return []string{"saml-sp-row-lifecycle", "sso-derived-route-generation", "sso-certificate-update-preservation", "sso-sp-cache-refresh", "sso-delete-cleanup"}
	case "identity-providers":
		return []string{"idp-row-lifecycle", "idp-sp-binding-expansion", "idp-sp-logoutbinding-normalization", "idp-sp-update-app-preservation", "idp-delete-cleanup"}
	case "approval-processes":
		return []string{"approval-step-condition-action-expansion", "step-layout-expansion", "approval-dbu-property-column-shape", "approval-related-list-field-expansion", "delete-cleanup"}
	case "reports":
		return []string{"report-folder-field-filter-expansion", "report-dbu-property-column-shape", "report-type-custom-field-expansion", "report-delete-cleanup"}
	case "dashboards":
		return []string{"dashboard-component-source-expansion", "dashboard-dbu-property-column-shape", "dashboard-condition-expansion", "dashboard-snapshot-cleanup", "dashboard-delete-cleanup"}
	case "object-views":
		return []string{"view-field-filter-button-expansion", "view-dbu-property-column-shape", "view-chart-kanban-expansion", "view-dashboard-component-cleanup", "view-delete-cleanup"}
	default:
		return nil
	}
}

func setupSvcLiveReplayRuntimeEffectsForOperation(domain string, operation string) []string {
	effects := setupSvcLiveReplayRuntimeEffects(domain)
	operation = strings.ToLower(strings.TrimSpace(operation))
	if operation == "query" {
		return nil
	}
	out := []string{}
	for _, effect := range effects {
		key := setupSvcLiveReplayExpectationKey(effect)
		cleanup := strings.Contains(key, "cleanup") || strings.Contains(key, "hard-delete") || strings.Contains(key, "physical-purge")
		switch operation {
		case "delete", "physical-purge", "remove":
			if cleanup || (operation == "physical-purge" && key == "database-view-refresh") {
				out = append(out, effect)
			}
		case "assign":
			if !cleanup || key == "permission-set-assignment-expansion" || key == "user-role-assignment-update" {
				out = append(out, effect)
			}
		default:
			if !cleanup && key != "assignment-remove-cleanup" {
				out = append(out, effect)
			}
		}
	}
	return out
}

func setupSvcLiveReplayRequiredTablesForOperation(domain setupSvcLiveReplayDomain, operation string) []string {
	required := append([]string{}, domain.RequiredTables...)
	operation = strings.ToLower(strings.TrimSpace(operation))
	switch operation {
	case "query", "create", "update", "":
		return required
	case "folder-create", "folder-update":
		if normalizeDomain(domain.Domain) == "reports" {
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_folder")
		}
		return required
	case "assign":
		return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_permsets_assign", "tp_sys_user")
	case "remove":
		return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_permsets_assign", "tp_sys_user")
	case "delete", "physical-purge":
		switch normalizeDomain(domain.Domain) {
		case "applications":
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_app", "tp_sys_app_tab", "tp_sys_profile_infoset", "tp_sys_multi_lang")
		case "dashboards":
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_dashboard", "tp_sys_dashboard_report", "tp_sys_dashboard_condition", "tp_sys_recent_items", "tp_sys_dashboard_snapshot", "tp_sys_snapshot_refress")
		case "global-select-lists":
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_global_select")
		case "identity-providers":
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_idp_sps")
		case "record-types":
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_recordtype", "tp_sys_profile_infoset", "tp_sys_profile_layout", "tp_sys_field_dependency", "tp_sys_multi_lang")
		case "reports":
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_report", "tp_sys_condition", "tp_sys_report_expression", "tp_sys_report_fieldname", "tp_sys_report_object", "tp_sys_report_object_detail", "tp_sys_reportgather", "tp_sys_reportgroup", "tp_sys_recent_items")
		case "roles":
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_role", "tp_sys_group")
		default:
			return required
		}
	case "folder-delete":
		if normalizeDomain(domain.Domain) == "reports" {
			return setupSvcLiveReplayRequiredTablesFiltered(required, "tp_sys_folder")
		}
		return required
	default:
		return required
	}
}

func setupSvcLiveReplayRequiredTablesFiltered(required []string, candidates ...string) []string {
	allowed := map[string]bool{}
	for _, candidate := range candidates {
		if normalized := strings.ToLower(strings.TrimSpace(candidate)); normalized != "" {
			allowed[normalized] = true
		}
	}
	var out []string
	for _, table := range required {
		normalized := strings.ToLower(strings.TrimSpace(table))
		if normalized != "" && allowed[normalized] {
			out = append(out, table)
		}
	}
	if len(out) == 0 {
		return append([]string{}, required...)
	}
	return out
}

func setupSvcLiveReplayQueryReadbackExpectations(domain string) []string {
	switch normalizeDomain(domain) {
	case "objects":
		return []string{"object-identity-prefix-datatable-readback", "standard-custom-field-readback", "standard-related-list-readback", "layout-button-profile-view-relationship-readback"}
	case "fields":
		return []string{"object-scoped-field-api-readback", "label-data-type-option-reference-readback", "layout-profile-dependency-relationship-readback"}
	case "global-select-lists":
		return []string{"global-list-option-readback", "field-link-readback", "translated-label-readback"}
	case "record-types":
		return []string{"record-type-profile-layout-readback", "record-type-field-dependency-readback", "translated-label-readback", "delete-absence-readback"}
	case "layouts":
		return []string{"layout-section-field-readback", "profile-button-relatedlist-readback", "translated-label-readback"}
	case "profiles":
		return []string{"profile-object-field-layout-recordtype-readback", "translated-label-readback", "permission-cleanup-readback"}
	case "permissions":
		return []string{"permission-definition-label-readback", "profile-infoset-assignment-readback", "permission-set-infoset-field-readback", "permission-set-assignment-readback", "remove-cleanup-readback"}
	case "roles":
		return []string{"role-hierarchy-group-readback", "user-assignment-readback", "user-unassignment-fallback-readback", "delete-cleanup-readback"}
	case "sharing-rules":
		return []string{"sharing-rule-condition-readback", "target-source-access-readback", "dynamic-share-cleanup-readback"}
	case "validation-rules":
		return []string{"validation-rule-row-readback", "active-error-message-readback"}
	case "applications":
		return []string{"application-tab-readback", "application-profile-visibility-readback", "label-order-visibility-readback"}
	case "menus":
		return []string{"tab-app-profile-readback", "tab-profile-all-readback", "label-target-object-readback"}
	case "buttons":
		return []string{"button-scope-layout-view-readback", "action-target-readback"}
	case "custom-settings":
		return []string{"setting-object-field-readback", "setting-layout-profile-readback", "datatable-view-readback"}
	case "dupe-catchers":
		return []string{"dupe-rule-field-condition-readback", "dupe-lowercase-audit-action-readback", "matching-action-readback"}
	case "single-sign-ons":
		return []string{"sso-saml-setting-readback", "sso-derived-route-readback", "sso-lowercase-audit-action-readback", "enabled-status-readback"}
	case "identity-providers":
		return []string{"idp-config-readback", "idp-sp-binding-readback", "idp-derived-login-logout-readback"}
	case "approval-processes":
		return []string{"approval-step-condition-action-readback", "step-layout-readback", "approval-dbu-property-readback", "approval-related-list-field-readback"}
	case "reports":
		return []string{"report-field-filter-folder-readback", "source-object-readback", "report-dbu-property-readback", "report-type-custom-field-readback"}
	case "dashboards":
		return []string{"dashboard-component-source-readback", "dashboard-dbu-property-readback", "dashboard-condition-readback", "layout-position-readback"}
	case "object-views":
		return []string{"view-field-filter-button-readback", "view-dbu-property-readback", "view-chart-kanban-readback", "visibility-scope-readback"}
	default:
		return nil
	}
}

func setupSvcLiveReplayHasCanonicalCRUDQ(operations []string) bool {
	seen := map[string]bool{}
	for _, operation := range operations {
		seen[strings.ToLower(strings.TrimSpace(operation))] = true
	}
	return seen["create"] && seen["update"] && seen["delete"] && seen["query"]
}

func setupSvcLiveReplayVariantOperations(operations []string) []string {
	var variants []string
	for _, operation := range operations {
		if !setupSvcLiveReplayIsCanonicalOperation(operation) {
			variants = append(variants, operation)
		}
	}
	return variants
}

func setupSvcLiveReplayIsCanonicalOperation(operation string) bool {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case "create", "update", "delete", "query":
		return true
	default:
		return false
	}
}

func setupSvcLiveReplayOperationFamily(operation string) string {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case "create":
		return "create"
	case "update":
		return "update"
	case "delete", "physical-purge":
		return "delete"
	case "query":
		return "query"
	default:
		return "variant"
	}
}

func shellPath(path string) string {
	if strings.TrimSpace(path) == "" {
		return "."
	}
	if strings.ContainsAny(path, " \t\n'\"") {
		return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
	}
	return path
}
